package planner

import (
	"strings"
	"testing"

	"github.com/kubeflow/semantic-operator/internal/governance"
)

// finalSelect isolates the combine step, which follows the last CTE.
func finalSelect(t *testing.T, sql string) string {
	t.Helper()
	i := strings.LastIndex(sql, "\n)\nSELECT ")
	if i < 0 {
		t.Fatalf("no final SELECT after the CTEs:\n%s", sql)
	}
	return sql[i+3:]
}

// A join would drop every group that one part lacks.
func TestCompositeStacksPartsWithoutJoins(t *testing.T) {
	cm := compiled(t)
	plan, err := Build(cm, testDialect(t), Request{
		Metrics:    []string{"total_sales", "store_productivity"},
		Dimensions: []string{"item.i_category"},
	}, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	sql := plan.SQL
	final := finalSelect(t, sql)
	if strings.Contains(final, "JOIN") || strings.Contains(sql, "<=>") {
		t.Fatalf("combine step must not join the parts:\n%s", sql)
	}
	if strings.Contains(sql, "COALESCE") {
		t.Fatalf("a missing side must stay NULL:\n%s", sql)
	}
	want := "SELECT `d0` AS `item.i_category`,\n" +
		"       MAX(`base_total_sales`) AS `total_sales`,\n" +
		"       (MAX(`num_store_productivity`) / NULLIF(MAX(`den_store_productivity`), 0)) AS `store_productivity`\n" +
		"FROM `stacked`\n" +
		"GROUP BY 1"
	if !strings.HasPrefix(final, want) {
		t.Fatalf("unexpected combine step, want prefix:\n%s\ngot:\n%s", want, final)
	}
	// Trino inlines CTEs, so a second reference would recompute the part.
	for _, part := range []string{"`base`", "`m_store_productivity_num`", "`m_store_productivity_den`"} {
		if n := strings.Count(sql, "FROM "+part); n != 1 {
			t.Errorf("part %s referenced %d times, want 1:\n%s", part, n, sql)
		}
	}
	for _, branch := range []string{
		"SELECT `d0` AS `d0`, `m_total_sales` AS `base_total_sales`, NULL AS `num_store_productivity`, NULL AS `den_store_productivity`\n  FROM `base`",
		"SELECT `d0` AS `d0`, NULL AS `base_total_sales`, `val` AS `num_store_productivity`, NULL AS `den_store_productivity`\n  FROM `m_store_productivity_num`",
		"SELECT `d0` AS `d0`, NULL AS `base_total_sales`, NULL AS `num_store_productivity`, `val` AS `den_store_productivity`\n  FROM `m_store_productivity_den`",
	} {
		if !strings.Contains(sql, branch) {
			t.Errorf("missing stacked branch:\n%s\nin:\n%s", branch, sql)
		}
	}
	if n := strings.Count(sql, "UNION ALL"); n != 2 {
		t.Errorf("want 2 UNION ALL for 3 parts, got %d:\n%s", n, sql)
	}
}

// A global aggregate returns one row even when a part is empty.
func TestCompositeWithoutDimensions(t *testing.T) {
	cm := compiled(t)
	plan, err := Build(cm, testDialect(t), Request{
		Metrics: []string{"store_productivity"},
	}, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	final := finalSelect(t, plan.SQL)
	want := "SELECT (MAX(`num_store_productivity`) / NULLIF(MAX(`den_store_productivity`), 0)) AS `store_productivity`\n" +
		"FROM `stacked`"
	if final != want {
		t.Fatalf("unexpected combine step without dimensions:\nwant:\n%s\ngot:\n%s", want, final)
	}
	if !strings.Contains(plan.SQL, "SELECT `val` AS `num_store_productivity`, NULL AS `den_store_productivity`") {
		t.Fatalf("branches must carry no dimension columns:\n%s", plan.SQL)
	}
}

// A repeated metric must not compute its sides twice.
func TestCompositeDeduplicatesRepeatedMetric(t *testing.T) {
	cm := compiled(t)
	plan, err := Build(cm, testDialect(t), Request{
		Metrics:    []string{"store_productivity", "store_productivity"},
		Dimensions: []string{"store.s_state"},
	}, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(plan.SQL, "`m_store_productivity_num` AS ("); n != 1 {
		t.Fatalf("numerator CTE emitted %d times, want 1:\n%s", n, plan.SQL)
	}
	if n := strings.Count(plan.SQL, "AS `store_productivity`"); n != 2 {
		t.Fatalf("want two output columns for the repeated metric, got %d:\n%s", n, plan.SQL)
	}
}

// Plan and result caches rely on byte-identical SQL.
func TestCompositeIsDeterministic(t *testing.T) {
	cm := compiled(t)
	req := Request{
		Metrics:    []string{"store_productivity", "total_sales"},
		Dimensions: []string{"item.i_category", "store.s_state"},
		MetricFilters: []MetricFilter{
			{Metric: "store_productivity", Op: ">", Value: 1},
		},
	}
	for _, dialect := range []string{"starrocks", "trino"} {
		d := testDialect(t)
		if dialect == "trino" {
			d = trinoDialect(t)
		}
		a, err := Build(cm, d, req, governance.Single("admin"))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			b, err := Build(cm, d, req, governance.Single("admin"))
			if err != nil {
				t.Fatal(err)
			}
			if a.SQL != b.SQL {
				t.Fatalf("%s: same request produced different SQL:\n%s\n---\n%s", dialect, a.SQL, b.SQL)
			}
		}
		if !strings.Contains(a.SQL, "GROUP BY 1, 2\nHAVING ") {
			t.Fatalf("%s: metric filter must follow the final GROUP BY:\n%s", dialect, a.SQL)
		}
	}
}

// A ratio over the one side is inline-safe until a later metric adds the
// fact table, which moves the join root and fans the ratio out.
func TestRatioClassifiedAgainstFinalJoinRoot(t *testing.T) {
	spec := testSpec()
	spec.Ossie.Metrics = append(spec.Ossie.Metrics,
		metric("employees_per_store", "SUM(store.s_number_employees) / NULLIF(COUNT(DISTINCT store.s_store_sk), 0)"))
	cm, err := Compile(spec, "semantic-system", "tpcds-retail")
	if err != nil {
		t.Fatal(err)
	}
	for _, metrics := range [][]string{
		{"employees_per_store", "total_sales"},
		{"employees_per_store", "total_sales", "employees_per_store"},
	} {
		plan, err := Build(cm, testDialect(t), Request{
			Metrics:    metrics,
			Dimensions: []string{"store.s_state"},
		}, governance.Single("admin"))
		if err != nil {
			t.Fatal(err)
		}
		final := finalSelect(t, plan.SQL)
		if strings.Contains(plan.SQL, "base_employees_per_store") {
			t.Fatalf("%v: ratio was computed inline over the fact join:\n%s", metrics, plan.SQL)
		}
		if n := strings.Count(final, "(MAX(`num_employees_per_store`) / NULLIF(MAX(`den_employees_per_store`), 0))"); n != strings.Count(strings.Join(metrics, ","), "employees_per_store") {
			t.Fatalf("%v: every output of the ratio must read the split sides:\n%s", metrics, plan.SQL)
		}
	}
}

// Plan.Columns lists every requested metric, so the SQL must too.
func TestRepeatedFlatMetricKeepsEveryColumn(t *testing.T) {
	cm := compiled(t)
	plan, err := Build(cm, testDialect(t), Request{
		Metrics:    []string{"total_sales", "total_sales"},
		Dimensions: []string{"store.s_state"},
	}, governance.Single("admin"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(plan.SQL, "AS `total_sales`"); n != len(plan.Columns)-1 {
		t.Fatalf("SQL has %d total_sales columns, Plan.Columns=%v:\n%s", n, plan.Columns, plan.SQL)
	}
}
