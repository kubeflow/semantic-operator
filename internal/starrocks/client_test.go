/*
Copyright The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package starrocks

import (
	"context"
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
