package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/planner"
	"github.com/kubeflow/semantic-operator/internal/serving"
)

// headerTransport adds the user and role headers to every request, the way
// the proxy in front of the server would.
type headerTransport struct{ role string }

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(PrincipalHeader, "tester")
	if h.role != "" {
		r.Header.Set(RoleHeader, h.role)
	}
	return http.DefaultTransport.RoundTrip(r)
}

// discoveryModel has ai_context on the model, on two metrics, and on one
// dimension. The analyst can't see payroll_cost, so its text is marked SECRET
// to catch any leak.
func discoveryModel() planner.CompiledModel {
	return planner.CompiledModel{
		Name: "retail", Version: "v1", Namespace: "ns", Resource: "retail",
		AIContext: v1alpha1.AIContext{
			Instructions: "Fiscal year starts in February.",
			Examples:     []string{"Revenue by state last quarter"},
		},
		MetricOrder: []string{"revenue", "payroll_cost"},
		Metrics: map[string]*planner.CompiledMetric{
			"revenue": {Name: "revenue", Raw: "SUM(s.amount)", AIContext: v1alpha1.AIContext{
				Instructions: "Use only for current-state analysis",
				Synonyms:     []string{"sales"},
				Examples:     []string{"What is our revenue?"},
			}},
			"payroll_cost": {Name: "payroll_cost", Raw: "SUM(e.salary)", AIContext: v1alpha1.AIContext{
				Instructions: "SECRET payroll guidance",
			}},
		},
		DatasetOrder: []string{"store"},
		Datasets: map[string]*planner.CompiledDataset{
			"store": {
				Name: "store", FieldOrder: []string{"s_state"},
				Fields: map[string]*planner.CompiledField{
					"s_state": {Name: "s_state", IsDimension: true, AIContext: v1alpha1.AIContext{
						Instructions: "Two-letter US state code.",
					}},
				},
			},
		},
		Governance: &v1alpha1.GovernanceSpec{
			DefaultRole: "analyst",
			Roles: []v1alpha1.RolePolicy{
				{Name: "analyst", AllowMetrics: []string{"revenue"}},
				{Name: "admin", AllowMetrics: []string{"*"}},
			},
		},
	}
}

// callTool calls one MCP tool over HTTP as the given role and decodes the
// result into out. It fails if any SECRET text shows up in the result.
func callTool(t *testing.T, role, tool string, args, out any) {
	t.Helper()
	blob, err := json.Marshal(discoveryModel())
	if err != nil {
		t.Fatal(err)
	}
	store := serving.NewStore()
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(&serving.Service{Store: store}, "test", nil, nil))
	defer srv.Close()

	ctx := context.Background()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{
		Endpoint:             srv.URL,
		HTTPClient:           &http.Client{Transport: headerTransport{role: role}},
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if res.IsError {
		t.Fatalf("%s returned a tool error: %+v", tool, res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "SECRET") {
		t.Fatalf("%s leaked AI context of a denied element: %s", tool, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("%s: decode %s: %v", tool, raw, err)
	}
}

func TestListModelsToolCarriesModelAIContext(t *testing.T) {
	var out listModelsOut
	callTool(t, "analyst", "list_models", map[string]any{}, &out)
	if len(out.Models) != 1 {
		t.Fatalf("want one model, got %+v", out.Models)
	}
	got := out.Models[0]
	if got.Instructions != "Fiscal year starts in February." ||
		!reflect.DeepEqual(got.Examples, []string{"Revenue by state last quarter"}) {
		t.Fatalf("list_models missing model ai_context, got %+v", got)
	}
}

func TestListMetricsToolCarriesAIContextAndHidesDenied(t *testing.T) {
	var out listMetricsOut
	callTool(t, "analyst", "list_metrics", map[string]any{"model": "retail"}, &out)
	want := []serving.MetricInfo{{
		Name:         "revenue",
		Instructions: "Use only for current-state analysis",
		Synonyms:     []string{"sales"},
		Examples:     []string{"What is our revenue?"},
	}}
	if !reflect.DeepEqual(out.Metrics, want) {
		t.Fatalf("list_metrics:\n got %+v\nwant %+v", out.Metrics, want)
	}
}

func TestListDimensionsToolCarriesAIContext(t *testing.T) {
	var out listDimensionsOut
	callTool(t, "admin", "list_dimensions", map[string]any{"model": "retail"}, &out)
	if len(out.Dimensions) != 1 || out.Dimensions[0].Instructions != "Two-letter US state code." {
		t.Fatalf("list_dimensions missing field ai_context, got %+v", out.Dimensions)
	}
}
