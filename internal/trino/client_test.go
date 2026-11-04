package trino

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/semantic-operator/internal/dbclient"
)

func TestDescribeQuery(t *testing.T) {
	q := describeQuery("iceberg", "osi_demo", "store_sales")
	want := `SELECT column_name, data_type FROM "iceberg".information_schema.columns ` +
		`WHERE table_schema = 'osi_demo' AND table_name = 'store_sales' ORDER BY ordinal_position`
	if q != want {
		t.Errorf("describeQuery =\n%q\nwant\n%q", q, want)
	}
}

func TestDescribeQueryEscapesLiterals(t *testing.T) {
	q := describeQuery(`ice"berg`, "o'demo", "ta'ble")
	if !strings.Contains(q, `"ice""berg"`) {
		t.Errorf("catalog identifier not escaped: %q", q)
	}
	if !strings.Contains(q, "'o''demo'") || !strings.Contains(q, "'ta''ble'") {
		t.Errorf("string literals not escaped: %q", q)
	}
}

// The Trino driver only accepts certificate trust through a registered custom
// client, so any TLS setting must select one. These tests stop before any dial.

func TestOpenUsesCustomClientWhenTLSIsRequested(t *testing.T) {
	c, err := Open(dbclient.Config{Host: "trino.example", Port: 8443, TLSEnabled: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if c.scheme != "https" {
		t.Errorf("scheme = %q, want https", c.scheme)
	}
	if c.customClient == "" {
		t.Error("TLS settings must be wired through a registered custom client")
	}
}

func TestOpenWithoutTLSKeepsPlainHTTP(t *testing.T) {
	c, err := Open(dbclient.Config{Host: "trino.example", Port: 8080})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if c.scheme != "http" || c.customClient != "" {
		t.Errorf("scheme = %q customClient = %q, want plain http with no custom client", c.scheme, c.customClient)
	}
}

func TestOpenRejectsUnreadableTLSCA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dbclient.Config{Host: "trino.example", TLSCAFile: path}); err == nil {
		t.Fatal("Open accepted a CA bundle that holds no certificate")
	}
}
