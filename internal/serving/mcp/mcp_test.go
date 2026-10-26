package mcp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kubeflow/semantic-operator/internal/planner"
)

func TestNewServerBuildsRecursiveFilterSchema(t *testing.T) {
	if NewServer(nil, "test", nil) == nil {
		t.Fatal("expected MCP server")
	}
}

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

func TestPlannerRequestMapsBooleanFilterGroups(t *testing.T) {
	body := []byte(`{"metrics":["total_sales"],"filter":{"or":[{"field":"store.s_state","op":"=","value":"NY"},{"and":[{"field":"store.s_store_sk","op":"IS NULL"},{"field":"store.s_state","op":"NOT LIKE","value":"TX%"}]}]}}`)
	var in queryIn
	if err := json.Unmarshal(body, &in); err != nil {
		t.Fatal(err)
	}
	got := in.plannerRequest()
	want := &planner.Filter{Or: []planner.Filter{
		{Field: "store.s_state", Op: "=", Value: "NY"},
		{And: []planner.Filter{
			{Field: "store.s_store_sk", Op: "IS NULL"},
			{Field: "store.s_state", Op: "NOT LIKE", Value: "TX%"},
		}},
	}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("filter tree mismatch:\ngot:  %#v\nwant: %#v", got.Filter, want)
	}
}
