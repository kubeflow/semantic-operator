package v1alpha1

import (
	"fmt"
	"strings"
)

// DialectPreference is the order in which expression dialects are selected
// for compilation. STARROCKS is an operator extension dialect; ANSI_SQL is
// the Ossie default and is passed through to StarRocks unchanged.
var DialectPreference = []string{"STARROCKS", "ANSI_SQL"}

// Select returns the expression body for the most preferred dialect present.
func (e Expression) Select() (string, bool) {
	for _, d := range DialectPreference {
		for _, de := range e.Dialects {
			if de.Dialect == d {
				return de.Expression, true
			}
		}
	}
	return "", false
}

// FindDataset returns the dataset with the given logical name.
func (m *OssieModel) FindDataset(name string) *Dataset {
	for i := range m.Datasets {
		if m.Datasets[i].Name == name {
			return &m.Datasets[i]
		}
	}
	return nil
}

// FindField returns the field with the given name.
func (d *Dataset) FindField(name string) *Field {
	for i := range d.Fields {
		if d.Fields[i].Name == name {
			return &d.Fields[i]
		}
	}
	return nil
}

// FindMetric returns the metric with the given name.
func (m *OssieModel) FindMetric(name string) *Metric {
	for i := range m.Metrics {
		if m.Metrics[i].Name == name {
			return &m.Metrics[i]
		}
	}
	return nil
}

// Role returns the policy for the named role.
func (g *GovernanceSpec) Role(name string) *RolePolicy {
	if g == nil {
		return nil
	}
	for i := range g.Roles {
		if g.Roles[i].Name == name {
			return &g.Roles[i]
		}
	}
	return nil
}

// FindMetricConstraint returns the constraint for the named metric, if any.
func (s *SemanticModelSpec) FindMetricConstraint(metric string) *MetricConstraint {
	for i := range s.MetricConstraints {
		if s.MetricConstraints[i].Metric == metric {
			return &s.MetricConstraints[i]
		}
	}
	return nil
}

// MatchDimensionPattern reports whether a "dataset.field" reference matches a
// metric constraint pattern, which is either "dataset.field" or "dataset.*".
func MatchDimensionPattern(pattern, ref string) bool {
	if ds, ok := strings.CutSuffix(pattern, ".*"); ok {
		return strings.HasPrefix(ref, ds+".")
	}
	return pattern == ref
}

// Refusal returns why the constraint stops its metric from being grouped by
// ref, or "" when it is allowed. A denied pattern wins over an allowed one.
func (c *MetricConstraint) Refusal(ref string) string {
	for _, p := range c.DeniedDimensions {
		if MatchDimensionPattern(p, ref) {
			return fmt.Sprintf("denied by metricConstraints pattern %q", p)
		}
	}
	if len(c.AllowedDimensions) == 0 {
		return ""
	}
	for _, p := range c.AllowedDimensions {
		if MatchDimensionPattern(p, ref) {
			return ""
		}
	}
	return "allowed only " + strings.Join(c.AllowedDimensions, ", ")
}
