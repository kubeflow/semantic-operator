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
package cache

import (
	"context"
	"strings"
	"testing"
)

func TestKeysScopeByDialectModelVersionAndRequest(t *testing.T) {
	a := PlanKey("starrocks", "m", "v1", "req1")
	b := PlanKey("starrocks", "m", "v2", "req1")
	c := PlanKey("starrocks", "m", "v1", "req2")
	d := PlanKey("trino", "m", "v1", "req1")
	if a == b || a == c {
		t.Fatal("plan keys must vary by model version and request hash")
	}
	if a == d {
		t.Fatal("plan keys must vary by dialect: switching engines must not serve the other engine's SQL")
	}
	if !strings.HasPrefix(a, "sl:plan:starrocks:m:v1:") {
		t.Fatalf("unexpected key shape: %s", a)
	}

	r1 := ResultKey("m", "v1", "SELECT 1")
	r2 := ResultKey("m", "v1", "SELECT 2")
	if r1 == r2 {
		t.Fatal("result keys must vary by SQL")
	}
	if PlanKey("starrocks", "m", "v1", "req1", "decision-a") ==
		PlanKey("starrocks", "m", "v1", "req1", "decision-b") {
		t.Fatal("plan keys must vary by external authorization decision")
	}
	if ResultKey("m", "v1", "SELECT 1", "decision-a") ==
		ResultKey("m", "v1", "SELECT 1", "decision-b") {
		t.Fatal("result keys must vary by external authorization decision")
	}
	if PlanKey("starrocks", "m", "v1", "req1", "") != a {
		t.Fatal("empty authorization fingerprint changed the legacy plan key")
	}
	if ResultKey("m", "v1", "SELECT 1", "") != r1 {
		t.Fatal("empty authorization fingerprint changed the legacy result key")
	}
}

func TestNilCacheIsNoOp(t *testing.T) {
	var c *Cache
	ctx := context.Background()
	if _, ok := c.GetPlan(ctx, "k"); ok {
		t.Fatal("nil cache must miss")
	}
	c.SetPlan(ctx, "k", []byte("v")) // must not panic
	if err := c.Ping(ctx); err != nil {
		t.Fatal("nil cache must be ready")
	}
}
