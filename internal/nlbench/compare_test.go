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
package nlbench

import (
	"testing"
	"time"
)

func TestCompareTimestampLabelsAcrossTransports(t *testing.T) {
	// Ground truth arrives from the direct MySQL client as time.Time values;
	// path answers arrive JSON-decoded as RFC3339 strings. The same instant
	// must compare equal (this was a false-WRONG on every time-grained row).
	jan := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2001, 2, 1, 0, 0, 0, 0, time.UTC)
	gtRows := [][]any{
		{jan, "8998813.87"},
		{feb, "8077528.14"},
	}
	res := PathResult{
		SQL: "SELECT ...",
		Rows: [][]any{
			{"2001-01-01T00:00:00Z", "8998813.87"},
			{"2001-02-01T00:00:00Z", "8077528.14"},
		},
	}
	if v := Compare([]string{"m", "total"}, gtRows, res); v != VerdictCorrect {
		t.Fatalf("timestamp-keyed rows across transports: got %s, want correct", v)
	}
}

func TestCompareBareDateEqualsDatetime(t *testing.T) {
	gtRows := [][]any{{"2001-01-01", "10.5"}}
	res := PathResult{SQL: "SELECT ...", Rows: [][]any{{"2001-01-01 00:00:00", "10.5"}}}
	if v := Compare(nil, gtRows, res); v != VerdictCorrect {
		t.Fatalf("bare date vs datetime: got %s, want correct", v)
	}
}

func TestCompareStillWrongOnDifferentNumbers(t *testing.T) {
	gtRows := [][]any{{"2001-01-01", "100.0"}}
	res := PathResult{SQL: "SELECT ...", Rows: [][]any{{"2001-01-01T00:00:00Z", "154705.51"}}}
	if v := Compare(nil, gtRows, res); v != VerdictWrong {
		t.Fatalf("different numbers must stay wrong, got %s", v)
	}
}

func TestCompareNonTemporalLabelsUntouched(t *testing.T) {
	gtRows := [][]any{{"Books", "10.0"}}
	res := PathResult{SQL: "SELECT ...", Rows: [][]any{{"Electronics", "10.0"}}}
	if v := Compare(nil, gtRows, res); v != VerdictWrong {
		t.Fatalf("label mismatch must stay wrong, got %s", v)
	}
}
