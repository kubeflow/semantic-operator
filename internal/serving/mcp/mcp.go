// Package mcp is the agent adapter: an MCP server on stateless streamable
// HTTP exposing list_metrics, list_dimensions, and query_metric. The LLM
// never writes SQL; it selects certified metrics and dimensions and the
// planner does the rest.
package mcp

import (
	"context"
	"net/http"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kubeflow/semantic-operator/internal/governance"
	"github.com/kubeflow/semantic-operator/internal/planner"
	"github.com/kubeflow/semantic-operator/internal/serving"
	"github.com/kubeflow/semantic-operator/internal/serving/auth"
)

// Identity headers are trusted only in header mode; see internal/serving/auth.
const (
	PrincipalHeader = auth.PrincipalHeader
	RoleHeader      = auth.RoleHeader
)

type listModelsIn struct{}

type listIn struct {
	Model string `json:"model,omitempty" jsonschema:"Semantic model name. Optional when exactly one model is published."`
}

type filterIn struct {
	Field  string           `json:"field,omitempty" jsonschema:"Modeled field reference as dataset.field. The field does not need a dimension declaration."`
	Op     string           `json:"op,omitempty" jsonschema:"One of = != < <= > >= IN NOT IN LIKE NOT LIKE BETWEEN IS NULL IS NOT NULL"`
	Value  any              `json:"value,omitempty" jsonschema:"Scalar comparison value"`
	Values []any            `json:"values,omitempty" jsonschema:"Values for IN / NOT IN / BETWEEN"`
	And    []filterLevel2In `json:"and,omitempty" jsonschema:"Filters that must all match"`
	Or     []filterLevel2In `json:"or,omitempty" jsonschema:"Filters where at least one must match"`
}

type filterLevel2In struct {
	Field  string           `json:"field,omitempty"`
	Op     string           `json:"op,omitempty"`
	Value  any              `json:"value,omitempty"`
	Values []any            `json:"values,omitempty"`
	And    []filterLevel3In `json:"and,omitempty"`
	Or     []filterLevel3In `json:"or,omitempty"`
}

type filterLevel3In struct {
	Field  string           `json:"field,omitempty"`
	Op     string           `json:"op,omitempty"`
	Value  any              `json:"value,omitempty"`
	Values []any            `json:"values,omitempty"`
	And    []filterLevel4In `json:"and,omitempty"`
	Or     []filterLevel4In `json:"or,omitempty"`
}

type filterLevel4In struct {
	Field  string           `json:"field,omitempty"`
	Op     string           `json:"op,omitempty"`
	Value  any              `json:"value,omitempty"`
	Values []any            `json:"values,omitempty"`
	And    []filterLevel5In `json:"and,omitempty"`
	Or     []filterLevel5In `json:"or,omitempty"`
}

type filterLevel5In struct {
	Field  string `json:"field,omitempty"`
	Op     string `json:"op,omitempty"`
	Value  any    `json:"value,omitempty"`
	Values []any  `json:"values,omitempty"`
}

type orderByIn struct {
	Field     string `json:"field" jsonschema:"Requested certified metric or dimension to order by"`
	Direction string `json:"direction" jsonschema:"Sort direction: asc or desc"`
}

type queryIn struct {
	Model         string                 `json:"model,omitempty" jsonschema:"Semantic model name. Optional when exactly one model is published."`
	Metrics       []string               `json:"metrics" jsonschema:"Certified metric names from list_metrics. At least one."`
	Dimensions    []string               `json:"dimensions,omitempty" jsonschema:"Explicitly declared group-by dimensions as dataset.field from list_dimensions"`
	Filters       []filterIn             `json:"filters,omitempty" jsonschema:"Row filters applied before aggregation"`
	Filter        *filterIn              `json:"filter,omitempty" jsonschema:"One row filter or a bounded AND/OR group"`
	MetricFilters []planner.MetricFilter `json:"metricFilters,omitempty" jsonschema:"Filters on requested certified metric results applied after aggregation"`
	Grain         string                 `json:"grain,omitempty" jsonschema:"Time grain: day, week, month, quarter, or year. Requires a time dimension in dimensions."`
	OrderBy       []orderByIn            `json:"orderBy,omitempty" jsonschema:"Ordered requested fields. Add every requested dimension after a metric to guarantee stable Top-N ties."`
	Limit         int                    `json:"limit,omitempty" jsonschema:"Row limit applied after metric filtering and ordering"`
}

func (in queryIn) plannerRequest() planner.Request {
	req := planner.Request{
		Metrics:       in.Metrics,
		Dimensions:    in.Dimensions,
		MetricFilters: in.MetricFilters,
		Filter:        toPlannerFilter(in.Filter),
		TimeGrain:     in.Grain,
		Limit:         in.Limit,
	}
	for _, f := range in.Filters {
		req.Filters = append(req.Filters, *toPlannerFilter(&f))
	}
	for _, order := range in.OrderBy {
		req.OrderBy = append(req.OrderBy, planner.OrderByClause{
			Field: order.Field, Direction: order.Direction,
		})
	}
	return req
}

func toPlannerFilter(in *filterIn) *planner.Filter {
	if in == nil {
		return nil
	}
	out := &planner.Filter{
		Field: in.Field, Op: in.Op, Value: in.Value, Values: in.Values,
	}
	for _, child := range in.And {
		out.And = append(out.And, toPlannerFilter2(child))
	}
	for _, child := range in.Or {
		out.Or = append(out.Or, toPlannerFilter2(child))
	}
	return out
}

func toPlannerFilter2(in filterLevel2In) planner.Filter {
	out := planner.Filter{Field: in.Field, Op: in.Op, Value: in.Value, Values: in.Values}
	for _, child := range in.And {
		out.And = append(out.And, toPlannerFilter3(child))
	}
	for _, child := range in.Or {
		out.Or = append(out.Or, toPlannerFilter3(child))
	}
	return out
}

func toPlannerFilter3(in filterLevel3In) planner.Filter {
	out := planner.Filter{Field: in.Field, Op: in.Op, Value: in.Value, Values: in.Values}
	for _, child := range in.And {
		out.And = append(out.And, toPlannerFilter4(child))
	}
	for _, child := range in.Or {
		out.Or = append(out.Or, toPlannerFilter4(child))
	}
	return out
}

func toPlannerFilter4(in filterLevel4In) planner.Filter {
	out := planner.Filter{Field: in.Field, Op: in.Op, Value: in.Value, Values: in.Values}
	for _, child := range in.And {
		out.And = append(out.And, toPlannerFilter5(child))
	}
	for _, child := range in.Or {
		out.Or = append(out.Or, toPlannerFilter5(child))
	}
	return out
}

func toPlannerFilter5(in filterLevel5In) planner.Filter {
	return planner.Filter{Field: in.Field, Op: in.Op, Value: in.Value, Values: in.Values}
}

type queryOut struct {
	Columns                  []string `json:"columns"`
	Rows                     [][]any  `json:"rows"`
	RowCount                 int      `json:"rowCount"`
	SQL                      string   `json:"sql"`
	Model                    string   `json:"model"`
	ModelVersion             string   `json:"modelVersion"`
	RequestHash              string   `json:"requestHash"`
	AuthorizationFingerprint string   `json:"authorizationFingerprint,omitempty"`
}

type listMetricsOut struct {
	Model   string               `json:"model"`
	Metrics []serving.MetricInfo `json:"metrics"`
}

type listDimensionsOut struct {
	Model      string                  `json:"model"`
	Dimensions []serving.DimensionInfo `json:"dimensions"`
}

type listModelsOut struct {
	Models []serving.ModelInfo `json:"models"`
}

// NewServer builds the MCP server over the shared service. The resolver turns
// the caller carried on the request context into the engine credential.
func NewServer(svc *serving.Service, version string, resolver serving.CredentialResolver) *sdk.Server {
	srv := sdk.NewServer(&sdk.Implementation{
		Name:    "ossie-semantic-layer",
		Title:   "Ossie Semantic Layer",
		Version: version,
	}, nil)

	sdk.AddTool(srv, &sdk.Tool{
		Name:        "list_models",
		Description: "List published semantic models.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, _ listModelsIn) (*sdk.CallToolResult, listModelsOut, error) {
		return nil, listModelsOut{Models: svc.Models(serving.IdentityFrom(ctx))}, nil
	})

	sdk.AddTool(srv, &sdk.Tool{
		Name: "list_metrics",
		Description: "List certified metrics with descriptions and synonyms. " +
			"Always ground the user's vocabulary here before querying: a phrase like " +
			"'revenue' or 'CLV' maps to exactly one certified metric.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listMetricsOut, error) {
		m, err := svc.Resolve(in.Model, serving.IdentityFrom(ctx))
		if err != nil {
			return nil, listMetricsOut{}, err
		}
		metrics, err := svc.ListMetrics(m, serving.IdentityFrom(ctx))
		if err != nil {
			return nil, listMetricsOut{}, err
		}
		return nil, listMetricsOut{Model: m.Name, Metrics: metrics}, nil
	})

	sdk.AddTool(srv, &sdk.Tool{
		Name: "list_dimensions",
		Description: "List explicitly declared group-by dimensions (dataset.field) with " +
			"types, synonyms, and time-dimension flags.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listDimensionsOut, error) {
		m, err := svc.Resolve(in.Model, serving.IdentityFrom(ctx))
		if err != nil {
			return nil, listDimensionsOut{}, err
		}
		dims, err := svc.ListDimensions(m, serving.IdentityFrom(ctx))
		if err != nil {
			return nil, listDimensionsOut{}, err
		}
		return nil, listDimensionsOut{Model: m.Name, Dimensions: dims}, nil
	})

	sdk.AddTool(srv, &sdk.Tool{
		Name: "query_metric",
		Description: "Query one or more certified metrics grouped by dimensions with " +
			"optional row filters (including bounded AND/OR groups and null checks), post-aggregation metric filters, time grain, ordering, and a row limit. " +
			"Metric filters may reference only requested certified metrics. For highest, lowest, " +
			"or Top-N questions, order by the requested metric and add every requested " +
			"dimension to guarantee stable ties before setting the limit. The semantic planner compiles " +
			"the request into deterministic, governed SQL and executes it on the configured " +
			"query engine. The response includes the SQL for provenance. Do not write SQL yourself.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, in queryIn) (*sdk.CallToolResult, queryOut, error) {
		m, err := svc.Resolve(in.Model, serving.IdentityFrom(ctx))
		if err != nil {
			return nil, queryOut{}, err
		}
		preq := in.plannerRequest()
		caller := serving.CallerFrom(ctx)
		cred, err := resolver(ctx, caller.Token, caller.EngineUser, caller.Expiry)
		if err != nil {
			return nil, queryOut{}, err
		}
		res, err := svc.Query(ctx, "mcp", m, preq, serving.IdentityFrom(ctx), cred)
		if err != nil {
			return nil, queryOut{}, err
		}
		return nil, queryOut{
			Columns: res.Columns, Rows: res.Rows, RowCount: res.RowCount,
			SQL: res.SQL, Model: res.Model, ModelVersion: res.ModelVersion,
			RequestHash: res.RequestHash, AuthorizationFingerprint: res.AuthorizationFingerprint,
		}, nil
	})

	return srv
}

// Handler wraps the MCP server in the stateless streamable HTTP transport and
// injects the resolved caller identity into the request context. A request the
// authenticator rejects never reaches the MCP server, so an agent cannot call
// a tool without a verified identity. A nil authenticator falls back to header
// mode so tests and embedders keep working.
func Handler(svc *serving.Service, version string, authn *auth.Authenticator, resolver serving.CredentialResolver) http.Handler {
	srv := NewServer(svc, version, resolver)
	h := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return srv },
		&sdk.StreamableHTTPOptions{Stateless: true},
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ac auth.Authenticated
		if authn == nil {
			principal := strings.TrimSpace(r.Header.Get(PrincipalHeader))
			if principal == "" {
				http.Error(w, "unauthenticated: no "+PrincipalHeader+" header", http.StatusUnauthorized)
				return
			}
			ac.Identity = governance.Single(r.Header.Get(RoleHeader))
			ac.Identity.Principal = principal
		} else {
			var err error
			ac, err = authn.Authenticate(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
		}
		ctx := serving.WithIdentity(r.Context(), ac.Identity)
		ctx = serving.WithCaller(ctx, serving.Caller{
			Token: ac.Token, EngineUser: ac.EngineUser, Expiry: ac.Expiry,
		})
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
