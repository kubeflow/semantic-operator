package dbclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// TLSRequested reports whether the connection should use TLS at all. Any
// setting counts, so a deployment that names only a CA bundle or only a client
// certificate still gets TLS instead of a silent plaintext connection.
func (c Config) TLSRequested() bool {
	return c.TLSEnabled ||
		c.TLSInsecureSkipVerify ||
		c.TLSServerName != "" ||
		c.TLSCAFile != "" ||
		c.TLSCertFile != "" ||
		c.TLSKeyFile != ""
}

// TLSConfig builds the tls.Config for the engine connection. A connection that
// asks for no TLS returns (nil, nil), which every client reads as "use the
// plaintext transport". Material is read once, at startup, so a missing or
// malformed file fails the process instead of the first query.
//
// The returned config is a fresh value per call: clients clone it per
// connection where they need to, and sharing one instance across pools would
// make that unsafe.
func (c Config) TLSConfig() (*tls.Config, error) {
	if !c.TLSRequested() {
		return nil, nil
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return nil, errors.New("engine TLS: certFile and keyFile must be set together")
	}
	cfg := &tls.Config{
		ServerName: c.TLSServerName,
		// TLS 1.2 is the floor both supported engines offer. Anything older is
		// not worth negotiating with a database.
		MinVersion: tls.VersionTLS12,
		// Opt-in for a self-signed development engine only.
		InsecureSkipVerify: c.TLSInsecureSkipVerify, //nolint:gosec // explicit development escape hatch, documented at the flag
	}
	if cfg.ServerName == "" {
		cfg.ServerName = c.Host
	}
	if c.TLSCAFile != "" {
		pemBytes, err := os.ReadFile(c.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("engine TLS: read caFile: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			// A scratch container may have no system bundle. A private CA is
			// then the whole trust store, which is a valid configuration.
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("engine TLS: caFile %q holds no PEM certificate", c.TLSCAFile)
		}
		cfg.RootCAs = pool
	}
	if c.TLSCertFile != "" {
		cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("engine TLS: load client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}
