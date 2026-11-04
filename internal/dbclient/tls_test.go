package dbclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testCert is a generated certificate with the pieces the TLS config needs.
type testCert struct {
	leaf   *x509.Certificate
	priv   *ecdsa.PrivateKey
	certPE []byte
	keyPEM []byte
}

// generateCert issues a self-signed CA when parent is nil, otherwise a leaf
// signed by that CA. Everything is ECDSA P-256 so the test stays fast.
func generateCert(t *testing.T, cn string, isCA bool, parent *testCert) testCert {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		tmpl.KeyUsage |= x509.KeyUsageCertSign
	} else {
		tmpl.DNSNames = []string{"engine.example"}
	}
	signerCert, signerKey := &tmpl, priv
	if parent != nil {
		signerCert, signerKey = parent.leaf, parent.priv
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, signerCert, &priv.PublicKey, signerKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return testCert{
		leaf:   leaf,
		priv:   priv,
		certPE: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		keyPEM: pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	}
}

// writePEM stores content under dir/name and returns the path.
func writePEM(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTLSRequested(t *testing.T) {
	cases := map[string]struct {
		cfg  Config
		want bool
	}{
		"nothing set": {Config{Host: "engine"}, false},
		"enabled":     {Config{TLSEnabled: true}, true},
		"skip verify": {Config{TLSInsecureSkipVerify: true}, true},
		"server name": {Config{TLSServerName: "engine.example"}, true},
		"ca file":     {Config{TLSCAFile: "/etc/ca.pem"}, true},
		"client cert": {Config{TLSCertFile: "/etc/tls.crt"}, true},
		"client key":  {Config{TLSKeyFile: "/etc/tls.key"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := tc.cfg.TLSRequested(); got != tc.want {
				t.Errorf("TLSRequested() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTLSConfigNotRequestedIsNil(t *testing.T) {
	cfg, err := (Config{Host: "engine"}).TLSConfig()
	if err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}
	if cfg != nil {
		t.Errorf("TLSConfig = %+v, want nil for a plaintext connection", cfg)
	}
}

func TestTLSConfigRequiresCertAndKeyTogether(t *testing.T) {
	for name, cfg := range map[string]Config{
		"cert without key": {TLSCertFile: "/etc/tls.crt"},
		"key without cert": {TLSKeyFile: "/etc/tls.key"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := cfg.TLSConfig(); err == nil {
				t.Fatal("TLSConfig accepted a certificate without its key")
			}
		})
	}
}

func TestTLSConfigDefaultsServerNameToHost(t *testing.T) {
	cfg, err := (Config{Host: "engine.internal", TLSEnabled: true}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "engine.internal" {
		t.Errorf("ServerName = %q, want the engine host", cfg.ServerName)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want TLS 1.2", cfg.MinVersion)
	}
	if cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify must stay false unless asked for")
	}
}

func TestTLSConfigServerNameOverride(t *testing.T) {
	cfg, err := (Config{Host: "10.0.0.7", TLSServerName: "engine.example"}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "engine.example" {
		t.Errorf("ServerName = %q, want the override", cfg.ServerName)
	}
}

func TestTLSConfigLoadsPrivateCA(t *testing.T) {
	dir := t.TempDir()
	ca := generateCert(t, "test-ca", true, nil)
	leaf := generateCert(t, "engine", false, &ca)
	caPath := writePEM(t, dir, "ca.pem", ca.certPE)

	cfg, err := (Config{Host: "engine.internal", TLSCAFile: caPath}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs is nil, the CA bundle was ignored")
	}
	// The whole point of the bundle is that the engine certificate verifies
	// against it, so check the chain rather than the field's presence.
	if _, err := leaf.leaf.Verify(x509.VerifyOptions{Roots: cfg.RootCAs, DNSName: "engine.example"}); err != nil {
		t.Errorf("engine certificate did not verify against the configured CA: %v", err)
	}
}

func TestTLSConfigLoadsClientCertificate(t *testing.T) {
	dir := t.TempDir()
	ca := generateCert(t, "test-ca", true, nil)
	client := generateCert(t, "semantic-server", false, &ca)
	certPath := writePEM(t, dir, "tls.crt", client.certPE)
	keyPath := writePEM(t, dir, "tls.key", client.keyPEM)

	cfg, err := (Config{TLSCertFile: certPath, TLSKeyFile: keyPath}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("Certificates = %d, want the configured client certificate", len(cfg.Certificates))
	}
}

func TestTLSConfigRejectsUnusableMaterial(t *testing.T) {
	dir := t.TempDir()
	garbage := writePEM(t, dir, "garbage.pem", []byte("not a certificate"))
	for name, cfg := range map[string]Config{
		"missing ca file":     {TLSCAFile: filepath.Join(dir, "absent.pem")},
		"ca file not pem":     {TLSCAFile: garbage},
		"missing client cert": {TLSCertFile: filepath.Join(dir, "absent.crt"), TLSKeyFile: filepath.Join(dir, "absent.key")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := cfg.TLSConfig(); err == nil {
				t.Fatal("TLSConfig accepted unusable material")
			}
		})
	}
}

func TestEnvConfigReadsTLSMaterial(t *testing.T) {
	t.Setenv("ENGINE_HOST", "engine.internal")
	t.Setenv("ENGINE_TLS_SERVER_NAME", "engine.example")
	t.Setenv("ENGINE_TLS_CA_FILE", "/etc/semantic/tls/ca.crt")
	t.Setenv("ENGINE_TLS_CERT_FILE", "/etc/semantic/tls/tls.crt")
	t.Setenv("ENGINE_TLS_KEY_FILE", "/etc/semantic/tls/tls.key")

	cfg, err := EnvConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLSServerName != "engine.example" ||
		cfg.TLSCAFile != "/etc/semantic/tls/ca.crt" ||
		cfg.TLSCertFile != "/etc/semantic/tls/tls.crt" ||
		cfg.TLSKeyFile != "/etc/semantic/tls/tls.key" {
		t.Errorf("EnvConfig TLS fields = %+v", cfg)
	}
}
