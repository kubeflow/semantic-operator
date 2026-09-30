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
package ossie

import (
	"testing"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
)

func TestValidateJoinsOK(t *testing.T) {
	spec := validSpec()
	spec.Joins = []v1alpha1.RelationshipJoin{{Relationship: "sales_to_customer", Type: "LEFT"}}
	if err := ValidateSpec(spec); err != nil {
		t.Fatalf("valid joins override rejected: %v", err)
	}
}

func TestValidateJoinsUnknownRelationship(t *testing.T) {
	spec := validSpec()
	spec.Joins = []v1alpha1.RelationshipJoin{{Relationship: "nope", Type: "LEFT"}}
	wantErr(t, spec, `relationship "nope" does not exist`)
}

func TestValidateJoinsDuplicateRejected(t *testing.T) {
	// Conflicting duplicates must be rejected, not resolved last-write-wins.
	spec := validSpec()
	spec.Joins = []v1alpha1.RelationshipJoin{
		{Relationship: "sales_to_customer", Type: "LEFT"},
		{Relationship: "sales_to_customer", Type: "INNER"},
	}
	wantErr(t, spec, `duplicate override for relationship "sales_to_customer"`)
}

func TestValidateJoinsBadType(t *testing.T) {
	spec := validSpec()
	spec.Joins = []v1alpha1.RelationshipJoin{{Relationship: "sales_to_date", Type: "FULL"}}
	wantErr(t, spec, `must be INNER or LEFT`)
}
