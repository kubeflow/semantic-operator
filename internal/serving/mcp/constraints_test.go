package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/emitter"
	_ "github.com/kubeflow/semantic-operator/internal/emitter/starrocks"
	"github.com/kubeflow/semantic-operator/internal/observability"
	"github.com/kubeflow/semantic-operator/internal/planner"
	"github.com/kubeflow/semantic-operator/internal/serving"
	"github.com/kubeflow/semantic-operator/internal/serving/rest"
)

// principalTransport adds the user header to every request, the way the proxy
// in front of the server would.
type principalTransport struct{}

func (principalTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(PrincipalHeader, "tester")
	return http.DefaultTransport.RoundTrip(r)
}

// constrainedService serves one model where revenue can't be grouped by
// store.s_state. No database is set: the request must fail before any SQL
// would run.
func constrainedService(t *testing.T) *serving.Service {
	t.Helper()
	f := func(name string, dim bool) v1alpha1.Field {
		fl := v1alpha1.Field{Name: name, Expression: v1alpha1.Expression{
			Dialects: []v1alpha1.DialectExpression{{Dialect: "ANSI_SQL", Expression: name}}}}
		if dim {
			fl.Dimension = &v1alpha1.Dimension{}
		}
		return fl
	}
	spec := &v1alpha1.SemanticModelSpec{
		Connection: v1alpha1.ConnectionSpec{Catalog: "iceberg", Database: "demo"},
		Ossie: v1alpha1.OssieModel{
			Name: "retail",
			Datasets: []v1alpha1.Dataset{{Name: "store", Source: "store", PrimaryKey: []string{"s_store_sk"},
				Fields: []v1alpha1.Field{f("s_store_sk", false), f("s_state", true)}}},
			Metrics: []v1alpha1.Metric{{Name: "revenue", Expression: v1alpha1.Expression{
				Dialects: []v1alpha1.DialectExpression{{Dialect: "ANSI_SQL", Expression: "SUM(store.s_store_sk)"}}}}},
		},
		MetricConstraints: []v1alpha1.MetricConstraint{
			{Metric: "revenue", DeniedDimensions: []string{"store.s_state"}},
		},
	}
	cm, err := planner.Compile(spec, "ns", "retail")
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(cm)
	if err != nil {
		t.Fatal(err)
	}
	store := serving.NewStore()
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}
	d, err := emitter.Get("starrocks")
	if err != nil {
		t.Fatal(err)
	}
	return &serving.Service{
		Store:   store,
		Dialect: d,
		Metrics: &observability.Metrics{
			Requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_requests_total", Help: "t"},
				[]string{"adapter", "model", "outcome"}),
		},
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tracer: otel.Tracer("test"),
	}
}

func TestRESTAndMCPEnforceTheSameMetricConstraint(t *testing.T) {
	svc := constrainedService(t)
	const want = `metric "revenue" cannot be grouped by "store.s_state"`

	// REST: a bad combination is a bad request.
	r := httptest.NewRequest(http.MethodPost, "/v1/models/retail/query",
		strings.NewReader(`{"metrics":["revenue"],"dimensions":["store.s_state"]}`))
	r.Header.Set(PrincipalHeader, "tester")
	w := httptest.NewRecorder()
	rest.Handler(svc, nil, serving.StaticResolver()).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("REST status = %d, want 400. body: %s", w.Code, w.Body.String())
	}
	var restErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &restErr); err != nil {
		t.Fatalf("REST body: %v", err)
	}
	if !strings.Contains(restErr.Error, want) {
		t.Fatalf("REST error = %q, want it to contain %q", restErr.Error, want)
	}

	// MCP: the same request comes back as a tool error with the same text.
	srv := httptest.NewServer(Handler(svc, "test", nil, serving.StaticResolver()))
	defer srv.Close()
	ctx := context.Background()
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx,
		&sdk.StreamableClientTransport{
			Endpoint:             srv.URL,
			HTTPClient:           &http.Client{Transport: principalTransport{}},
			DisableStandaloneSSE: true,
		}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = session.Close() }()
	res, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "query_metric", Arguments: map[string]any{
		"metrics": []string{"revenue"}, "dimensions": []string{"store.s_state"},
	}})
	if err != nil {
		t.Fatalf("query_metric: %v", err)
	}
	if !res.IsError || len(res.Content) == 0 {
		t.Fatalf("query_metric should fail, got %+v", res)
	}
	text, ok := res.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("unexpected MCP content: %#v", res.Content[0])
	}
	if text.Text != restErr.Error {
		t.Fatalf("REST and MCP disagree:\nREST: %s\nMCP:  %s", restErr.Error, text.Text)
	}
}
