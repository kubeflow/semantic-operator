package starrocks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/semantic-operator/internal/dbclient"
)

// The guards below all return before any network dial, so these tests need no
// live StarRocks. They lock in the passthrough preconditions.

func TestQueryRejectsExpiredToken(t *testing.T) {
	c, err := Open(Config{Host: "example.invalid", TLSEnabled: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	cred := dbclient.EngineCredential{
		Token:      "header.payload.sig",
		EngineUser: "alice",
		Expiry:     time.Now().Add(-time.Hour),
	}
	_, _, err = c.Query(context.Background(), cred, "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("want expiry error, got %v", err)
	}
}

func TestQueryRejectsMissingEngineUser(t *testing.T) {
	c, err := Open(Config{Host: "example.invalid", TLSEnabled: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	cred := dbclient.EngineCredential{Token: "header.payload.sig"} // no EngineUser
	_, _, err = c.Query(context.Background(), cred, "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "engine user") {
		t.Fatalf("want missing-engine-user error, got %v", err)
	}
}

func TestQueryRequiresTLSForPassthrough(t *testing.T) {
	c, err := Open(Config{Host: "example.invalid"}) // TLS disabled
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	cred := dbclient.EngineCredential{Token: "header.payload.sig", EngineUser: "alice"}
	_, _, err = c.Query(context.Background(), cred, "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "TLS") {
		t.Fatalf("want TLS-required error, got %v", err)
	}
}

func TestClientSupportsPerRequestIdentity(t *testing.T) {
	c, err := Open(Config{Host: "example.invalid", TLSEnabled: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if _, ok := any(c).(dbclient.PerRequestIdentityClient); !ok {
		t.Fatal("StarRocks client must implement dbclient.PerRequestIdentityClient")
	}
}

func TestOpenBuildsTLSConfigFromMaterial(t *testing.T) {
	c, err := Open(Config{Host: "starrocks.example", TLSEnabled: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if c.tlsConfig == nil {
		t.Fatal("TLS was requested but the client holds no tls.Config")
	}
	if c.tlsConfig.ServerName != "starrocks.example" {
		t.Errorf("ServerName = %q, want the engine host", c.tlsConfig.ServerName)
	}
}

func TestOpenWithoutTLSHasNoTLSConfig(t *testing.T) {
	c, err := Open(Config{Host: "starrocks.example"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if c.tlsConfig != nil {
		t.Errorf("tlsConfig = %+v, want nil for a plaintext connection", c.tlsConfig)
	}
}

func TestOpenRejectsUnreadableTLSCA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(Config{Host: "starrocks.example", TLSCAFile: path}); err == nil {
		t.Fatal("Open accepted a CA bundle that holds no certificate")
	}
}
