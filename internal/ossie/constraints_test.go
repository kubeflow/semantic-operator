package ossie

import (
	"testing"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
)

func TestValidateMetricConstraintsOK(t *testing.T) {
	s := validSpec()
	s.MetricConstraints = []v1alpha1.MetricConstraint{
		{Metric: "clv", DeniedDimensions: []string{"date_dim.*"}},
		{Metric: "total_sales", AllowedDimensions: []string{"date_dim.*"}, DeniedDimensions: []string{"date_dim.d_date"}},
	}
	if err := ValidateSpec(s); err != nil {
		t.Fatalf("valid constraints rejected: %v", err)
	}
}

func TestValidateMetricConstraintErrors(t *testing.T) {
	cases := []struct {
		name       string
		constraint v1alpha1.MetricConstraint
		want       string
	}{
		{
			name:       "unknown metric",
			constraint: v1alpha1.MetricConstraint{Metric: "nope", DeniedDimensions: []string{"date_dim.*"}},
			want:       `metricConstraints[0]: metric "nope" does not exist`,
		},
		{
			name:       "no patterns",
			constraint: v1alpha1.MetricConstraint{Metric: "clv"},
			want:       "set allowedDimensions or deniedDimensions",
		},
		{
			name:       "unknown dataset",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", DeniedDimensions: []string{"nope.*"}},
			want:       `dataset "nope" does not exist`,
		},
		{
			name:       "unknown field",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", AllowedDimensions: []string{"date_dim.nope"}},
			want:       `field "nope" not found on dataset "date_dim"`,
		},
		{
			name:       "field is not a dimension",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", DeniedDimensions: []string{"date_dim.d_date_sk"}},
			want:       `"date_dim.d_date_sk" is not a declared dimension`,
		},
		{
			name:       "no dataset",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", DeniedDimensions: []string{"date_dim"}},
			want:       `"date_dim" must be <dataset>.<field> or <dataset>.*`,
		},
		{
			name:       "partial wildcard",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", DeniedDimensions: []string{"date_dim.d_*"}},
			want:       `"date_dim.d_*" must be <dataset>.<field> or <dataset>.*`,
		},
		{
			name:       "bare wildcard",
			constraint: v1alpha1.MetricConstraint{Metric: "clv", AllowedDimensions: []string{"*"}},
			want:       `"*" must be <dataset>.<field> or <dataset>.*`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSpec()
			s.MetricConstraints = []v1alpha1.MetricConstraint{tc.constraint}
			wantErr(t, s, tc.want)
		})
	}
}

// Two entries for one metric would leave it unclear which rule wins.
func TestValidateDuplicateMetricConstraint(t *testing.T) {
	s := validSpec()
	s.MetricConstraints = []v1alpha1.MetricConstraint{
		{Metric: "clv", DeniedDimensions: []string{"date_dim.*"}},
		{Metric: "clv", AllowedDimensions: []string{"date_dim.d_year"}},
	}
	wantErr(t, s, `metricConstraints[1]: duplicate entry for metric "clv"`)
}

// A view that breaks a constraint would only fail later, when the operator
// tries to publish it. Catch it here instead.
func TestValidateViewBreaksMetricConstraint(t *testing.T) {
	s := validSpec()
	s.MetricConstraints = []v1alpha1.MetricConstraint{
		{Metric: "total_sales", DeniedDimensions: []string{"date_dim.*"}},
	}
	wantErr(t, s, `view "sales_by_year": metric "total_sales" cannot be grouped by "date_dim.d_year"`)
}
