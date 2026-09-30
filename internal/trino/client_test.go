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
package trino

import (
	"strings"
	"testing"
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
