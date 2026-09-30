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
package planner

import (
	"strings"
	"testing"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/governance"
)

// join_type is an operator extension delivered via spec.joins, not a field on
// the Ossie relationship. Compile should default to INNER and honor the override.
func TestJoinTypeOverride(t *testing.T) {
	d := testDialect(t)
	req := Request{
		Metrics:    []string{"total_sales"},
		Dimensions: []string{"customer.customer_full_name"},
	}

	// Default: INNER join to customer, no LEFT anywhere.
	cmDefault, err := Compile(testSpec(), "ns", "cr")
	if err != nil {
		t.Fatal(err)
	}
	planDefault, err := Build(cmDefault, d, req, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(planDefault.SQL, "INNER JOIN") || strings.Contains(planDefault.SQL, "LEFT JOIN") {
		t.Fatalf("default should INNER JOIN customer, got:\n%s", planDefault.SQL)
	}

	// Override sales_to_customer to LEFT via spec.joins.
	spec := testSpec()
	spec.Joins = []v1alpha1.RelationshipJoin{{Relationship: "sales_to_customer", Type: "LEFT"}}
	cmLeft, err := Compile(spec, "ns", "cr")
	if err != nil {
		t.Fatal(err)
	}
	planLeft, err := Build(cmLeft, d, req, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(planLeft.SQL, "LEFT JOIN") {
		t.Fatalf("override should produce LEFT JOIN, got:\n%s", planLeft.SQL)
	}

	found := false
	for _, r := range cmLeft.Relationships {
		if r.Name == "sales_to_customer" {
			found = true
			if r.JoinType != "LEFT" {
				t.Fatalf("compiled JoinType = %q, want LEFT", r.JoinType)
			}
		}
	}
	if !found {
		t.Fatal("sales_to_customer not found in compiled relationships")
	}
}
