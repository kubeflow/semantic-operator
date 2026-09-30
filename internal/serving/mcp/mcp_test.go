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
package mcp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kubeflow/semantic-operator/internal/planner"
)

func TestPlannerRequestMapsOrderBy(t *testing.T) {
	in := queryIn{
		Metrics:    []string{"total_sales"},
		Dimensions: []string{"item.i_category"},
		Filters: []filterIn{{
			Field: "store.s_state", Op: "IN", Values: []any{"NY", "CA"},
		}},
		MetricFilters: []planner.MetricFilter{{
			Metric: "total_sales", Op: "BETWEEN", Values: []any{100, 1000},
		}},
		Grain: "month",
		OrderBy: []orderByIn{
			{Field: "total_sales", Direction: "desc"},
			{Field: "item.i_category", Direction: "asc"},
		},
		Limit: 5,
	}

	got := in.plannerRequest()
	want := planner.Request{
		Metrics:    []string{"total_sales"},
		Dimensions: []string{"item.i_category"},
		Filters: []planner.Filter{{
			Field: "store.s_state", Op: "IN", Values: []any{"NY", "CA"},
		}},
		MetricFilters: []planner.MetricFilter{{
			Metric: "total_sales", Op: "BETWEEN", Values: []any{100, 1000},
		}},
		TimeGrain: "month",
		OrderBy: []planner.OrderByClause{
			{Field: "total_sales", Direction: "desc"},
			{Field: "item.i_category", Direction: "asc"},
		},
		Limit: 5,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("planner request mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestPlannerRequestPreservesMetricFilterJSONContract(t *testing.T) {
	body := []byte(`{"metrics":["total_sales"],"metricFilters":[{"metric":"total_sales","op":"BETWEEN","value":null,"values":[1,2]}]}`)
	var in queryIn
	if err := json.Unmarshal(body, &in); err != nil {
		t.Fatal(err)
	}
	var direct planner.Request
	if err := json.Unmarshal(body, &direct); err != nil {
		t.Fatal(err)
	}
	got := in.plannerRequest()
	if !reflect.DeepEqual(got.MetricFilters, direct.MetricFilters) {
		t.Fatalf("MCP metric filter contract differs from REST/planner decoding:\ngot:  %#v\nwant: %#v", got.MetricFilters, direct.MetricFilters)
	}
}
