//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"testing"
)

const (
	multiMetricDim     = "groups.label"
	multiMetricLimit   = 10000
	multiMetricRelTol  = 1e-9
	multiMetricAbsTol  = 1e-6
	multiMetricExample = 5
)

// A group missing on one side of a split ratio must not vanish silently.
// Default semantics: union of groups, NULL for a missing side, and no effect
// on other metrics in the request. Flat metrics give the expected values, so
// the test does not depend on fixture numbers.
func TestMultiMetricRequests(t *testing.T) {
	model := profiles[cfg.engine].multiMetricModel
	headers := bearer(token(t, "alice"))
	amount := metricByGroup(t, headers, model, "total_amount")
	weight := metricByGroup(t, headers, model, "total_weight")

	// Without groups that have no activity, the subtests prove nothing.
	idle := 0
	for g := range weight {
		if _, ok := amount[g]; !ok {
			idle++
		}
	}
	for g := range amount {
		if _, ok := weight[g]; !ok {
			t.Fatalf("fixture: group %q has activity but no groups row", g)
		}
	}
	if len(amount) == 0 || idle == 0 {
		t.Fatalf("fixture: want groups with and without activity, got %d with and %d without",
			len(amount), idle)
	}
	t.Logf("fixture: %d groups, %d with activity, %d without", len(weight), len(amount), idle)

	t.Run("amount per weight keeps groups without activity", func(t *testing.T) {
		want := map[string]*float64{}
		for g, den := range weight {
			want[g] = ratio(amount[g], den)
		}
		got := metricByGroup(t, headers, model, "amount_per_weight")
		assertGroups(t, "amount_per_weight", want, got)
	})

	t.Run("weight per amount keeps groups without activity", func(t *testing.T) {
		want := map[string]*float64{}
		for g, num := range weight {
			want[g] = ratio(num, amount[g])
		}
		got := metricByGroup(t, headers, model, "weight_per_amount")
		assertGroups(t, "weight_per_amount", want, got)
	})

	t.Run("total weight unchanged for every group when requested with a ratio", func(t *testing.T) {
		r := runMultiMetricQuery(t, headers, model, []string{"total_weight", "amount_per_weight"})
		if len(r.columns) != 3 || r.columns[1] != "total_weight" {
			t.Fatalf("unexpected columns %v", r.columns)
		}
		got := map[string]*float64{}
		for _, row := range r.rows {
			got[groupKey(t, row[0])] = cellFloat(t, row[1])
		}
		assertGroups(t, "total_weight with amount_per_weight", weight, got)
	})
}

// ratio mirrors NULLIF(den, 0) with no COALESCE.
func ratio(num, den *float64) *float64 {
	if num == nil || den == nil || *den == 0 {
		return nil
	}
	v := *num / *den
	return &v
}

func metricByGroup(t *testing.T, headers map[string]string, model, metric string) map[string]*float64 {
	t.Helper()
	r := runMultiMetricQuery(t, headers, model, []string{metric})
	if len(r.columns) != 2 || r.columns[0] != multiMetricDim || r.columns[1] != metric {
		t.Fatalf("%s: unexpected columns %v", metric, r.columns)
	}
	out := make(map[string]*float64, len(r.rows))
	for _, row := range r.rows {
		g := groupKey(t, row[0])
		if _, dup := out[g]; dup {
			t.Fatalf("%s: group %q returned more than once", metric, g)
		}
		out[g] = cellFloat(t, row[1])
	}
	return out
}

func runMultiMetricQuery(t *testing.T, headers map[string]string, model string, metrics []string) queryResult {
	t.Helper()
	body := map[string]any{
		"metrics":    metrics,
		"dimensions": []string{multiMetricDim},
		"orderBy":    []map[string]any{{"field": multiMetricDim, "direction": "asc"}},
		"limit":      multiMetricLimit,
	}
	r, err := queryModel(testCtx(t), cfg.staticNS, model, headers, body)
	if err != nil {
		t.Fatalf("query %v: %v", metrics, err)
	}
	if r.status != http.StatusOK {
		t.Fatalf("query %v: want 200, got %d (%s)", metrics, r.status, r.raw)
	}
	if len(r.rows) >= multiMetricLimit {
		t.Fatalf("query %v: result reached the limit of %d rows; the comparison would be partial", metrics, multiMetricLimit)
	}
	t.Logf("query %v: %d rows\n%s", metrics, len(r.rows), r.sql)
	return r
}

// assertGroups reports missing, extra, and wrong groups apart, so a failure
// shows which kind of bug it is.
func assertGroups(t *testing.T, label string, want, got map[string]*float64) {
	t.Helper()
	var missing, extra, differ []string
	for g, w := range want {
		v, ok := got[g]
		if !ok {
			missing = append(missing, g)
			continue
		}
		if !floatPtrClose(w, v) {
			differ = append(differ, fmt.Sprintf("%s: want %s, got %s", g, fmtPtr(w), fmtPtr(v)))
		}
	}
	for g := range got {
		if _, ok := want[g]; !ok {
			extra = append(extra, g)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%s: %d of %d groups are missing, e.g. %v", label, len(missing), len(want), examples(missing))
	}
	if len(extra) > 0 {
		t.Errorf("%s: %d unexpected groups, e.g. %v", label, len(extra), examples(extra))
	}
	if len(differ) > 0 {
		t.Errorf("%s: %d groups have a wrong value, e.g. %v", label, len(differ), examples(differ))
	}
}

func examples(s []string) []string {
	sort.Strings(s)
	if len(s) > multiMetricExample {
		return s[:multiMetricExample]
	}
	return s
}

// floatPtrClose also allows multiMetricAbsTol because StarRocks rounds
// DECIMAL division results to six decimal places.
func floatPtrClose(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return cellsClose(*a, *b, multiMetricRelTol) || math.Abs(*a-*b) <= multiMetricAbsTol
}

func fmtPtr(v *float64) string {
	if v == nil {
		return "NULL"
	}
	return strconv.FormatFloat(*v, 'g', -1, 64)
}

// groupKey accepts any scalar because engines decode date labels differently.
func groupKey(t *testing.T, cell any) string {
	t.Helper()
	if cell == nil {
		t.Fatal("group label is NULL; the fixtures have no NULL labels")
	}
	return fmt.Sprint(cell)
}

// cellFloat accepts decimal strings because engines return DECIMAL as text.
func cellFloat(t *testing.T, cell any) *float64 {
	t.Helper()
	switch v := cell.(type) {
	case nil:
		return nil
	case float64:
		return &v
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("metric cell %q is not numeric: %v", v, err)
		}
		return &n
	default:
		t.Fatalf("metric cell has unsupported type %T: %v", cell, cell)
		return nil
	}
}
