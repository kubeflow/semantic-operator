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
//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// Exchange mode swaps the caller's validated token for an engine-audience token
// (RFC 8693) under the same subject, so the engine session user is still the
// caller. These assertions prove the swapped token authenticates the OIDC user
// and that per-user enforcement holds through the exchange path.

// alice reads an allowed dimension through the exchanged token. A failed
// exchange would surface as an error, so a real 200 row proves the server
// obtained and used a valid engine-audience token for the caller.
func TestExchangeAllowedUserRealValue(t *testing.T) {
	tok := token(t, "alice")
	r, err := queryModel(testCtx(t), cfg.exchangeNS, cfg.model, bearer(tok), allowQuery())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if r.status != 200 {
		t.Fatalf("alice exchange: want 200, got %d (%s)", r.status, r.raw)
	}
	if len(r.rows) == 0 {
		t.Fatalf("alice exchange: no rows (%s)", r.raw)
	}
}

// bob is denied the denied table under the exchanged token, as under
// passthrough: the exchange preserves the subject, so the engine enforces bob's
// grants.
func TestExchangeDeniedTable(t *testing.T) {
	tok := token(t, "bob")
	r, err := queryModel(testCtx(t), cfg.exchangeNS, cfg.model, bearer(tok), denyQuery())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if !strings.Contains(strings.ToLower(r.errMsg), "denied") {
		t.Fatalf("bob exchange denied table: want an access-denied error, got status %d (%s)", r.status, r.raw)
	}
}

// bob still reads an allowed dimension through the exchanged token. This
// separates a table-specific denial from a blanket exchange failure: had the
// exchange failed closed, bob would be denied everywhere, not only the denied
// table.
func TestExchangeDeniedUserAllowedElsewhere(t *testing.T) {
	tok := token(t, "bob")
	r, err := queryModel(testCtx(t), cfg.exchangeNS, cfg.model, bearer(tok), allowQuery())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if r.status != 200 {
		t.Fatalf("bob exchange allowed table: want 200, got %d (%s)", r.status, r.raw)
	}
}
