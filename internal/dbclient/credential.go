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
package dbclient

import "time"

// EngineCredential is a per-request identity used to execute a query under the
// caller rather than the engine client's static credential. It carries the
// caller's validated access token for identity propagation.
//
// The zero value means "use the client's own credential", which is the static
// default. A non-empty Token is security sensitive: it must never be placed on
// a governance identity, written to logs or traces, embedded in a SQL comment,
// or mixed into any cache key.
type EngineCredential struct {
	// Token is the caller's validated bearer token, forwarded to the engine.
	Token string
	// EngineUser is the caller's engine session user, resolved from a
	// configured token claim. It is used as the engine session identity so
	// engine-side per-user policy applies to the real caller, and must match
	// the engine's own principal claim.
	EngineUser string
	// Expiry is the token's expiration, so execution can avoid using a token
	// past its lifetime. Zero means unknown.
	Expiry time.Time
}

// IsZero reports whether no credential is set, meaning static execution.
func (c EngineCredential) IsZero() bool { return c.Token == "" }
