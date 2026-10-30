package planner

import (
	"strings"
	"testing"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/emitter"
	"github.com/kubeflow/semantic-operator/internal/governance"
)

// constrainedSpec limits two metrics:
//   - total_profit can't be grouped by anything in date_dim.
//   - customer_lifetime_value can only be grouped by customer fields and
//     item.i_category, and never by customer.c_email_address.
func constrainedSpec() *v1alpha1.SemanticModelSpec {
	s := testSpec()
	s.MetricConstraints = []v1alpha1.MetricConstraint{
		{Metric: "total_profit", DeniedDimensions: []string{"date_dim.*"}},
		{
			Metric:            "customer_lifetime_value",
			AllowedDimensions: []string{"customer.*", "item.i_category"},
			DeniedDimensions:  []string{"customer.c_email_address"},
		},
	}
	return s
}

func compiledConstrained(t *testing.T) *CompiledModel {
	t.Helper()
	cm, err := Compile(constrainedSpec(), "ns", "cr")
	if err != nil {
		t.Fatal(err)
	}
	return cm
}

func TestMetricConstraints(t *testing.T) {
	cases := []struct {
		name    string
		metrics []string
		dims    []string
		wantErr string // empty means the request must plan
	}{
		{
			name:    "denied dataset",
			metrics: []string{"total_profit"},
			dims:    []string{"date_dim.d_year"},
			wantErr: `metric "total_profit" cannot be grouped by "date_dim.d_year": denied by metricConstraints pattern "date_dim.*"`,
		},
		{
			name:    "other dataset still works",
			metrics: []string{"total_profit"},
			dims:    []string{"item.i_category"},
		},
		{
			name:    "allowed field",
			metrics: []string{"customer_lifetime_value"},
			dims:    []string{"item.i_category", "customer.customer_full_name"},
		},
		{
			name:    "not in allowed list",
			metrics: []string{"customer_lifetime_value"},
			dims:    []string{"item.i_brand"},
			wantErr: `metric "customer_lifetime_value" cannot be grouped by "item.i_brand": allowed only customer.*, item.i_category`,
		},
		{
			name:    "deny wins over allow",
			metrics: []string{"customer_lifetime_value"},
			dims:    []string{"customer.c_email_address"},
			wantErr: `metric "customer_lifetime_value" cannot be grouped by "customer.c_email_address": denied by metricConstraints pattern "customer.c_email_address"`,
		},
		{
			name:    "one bad metric fails the whole request",
			metrics: []string{"total_sales", "total_profit"},
			dims:    []string{"date_dim.d_year"},
			wantErr: `metric "total_profit" cannot be grouped by "date_dim.d_year"`,
		},
		{
			name:    "unconstrained metric is unaffected",
			metrics: []string{"total_sales"},
			dims:    []string{"date_dim.d_year", "item.i_brand"},
		},
		{
			name:    "no dimensions",
			metrics: []string{"customer_lifetime_value"},
		},
	}
	cm := compiledConstrained(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Build(cm, testDialect(t), Request{Metrics: tc.metrics, Dimensions: tc.dims}, governance.Single("admin"))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want a plan, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want error %q, got SQL:\n%s", tc.wantErr, p.SQL)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v\nwant it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// Both engines must refuse the same request with the same message.
func TestMetricConstraintsSameOnEveryDialect(t *testing.T) {
	cm := compiledConstrained(t)
	req := Request{Metrics: []string{"total_profit"}, Dimensions: []string{"date_dim.d_year"}}
	var msgs []string
	for _, name := range []string{"starrocks", "trino"} {
		d, err := emitter.Get(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Build(cm, d, req, governance.Single("admin"))
		if err == nil {
			t.Fatalf("%s: invalid combination was planned", name)
		}
		msgs = append(msgs, err.Error())
	}
	if msgs[0] != msgs[1] {
		t.Fatalf("dialects disagree:\nstarrocks: %s\ntrino:     %s", msgs[0], msgs[1])
	}
}

// A caller who may not query the metric gets the normal 403 error, not a
// message about the metric's constraints.
func TestGovernanceRunsBeforeMetricConstraints(t *testing.T) {
	cm := compiledConstrained(t)
	_, err := Build(cm, testDialect(t), Request{
		Metrics: []string{"total_profit"}, Dimensions: []string{"date_dim.d_year"},
	}, governance.Single("stranger"))
	if err == nil || !strings.Contains(err.Error(), governance.ErrUnauthorized.Error()) {
		t.Fatalf("want an unauthorized error first, got %v", err)
	}
}

func TestCompileKeepsMetricConstraints(t *testing.T) {
	cm := compiledConstrained(t)
	c, ok := cm.MetricConstraints["customer_lifetime_value"]
	if !ok {
		t.Fatalf("constraint missing from compiled model: %+v", cm.MetricConstraints)
	}
	if strings.Join(c.AllowedDimensions, ",") != "customer.*,item.i_category" {
		t.Fatalf("allowed dimensions = %v", c.AllowedDimensions)
	}
	if len(compiled(t).MetricConstraints) != 0 {
		t.Fatal("a model without constraints should compile none")
	}
}

func TestMetricConstraintsChangeModelVersion(t *testing.T) {
	if SpecVersion(testSpec()) == SpecVersion(constrainedSpec()) {
		t.Fatal("adding metricConstraints must change the model version")
	}
}
