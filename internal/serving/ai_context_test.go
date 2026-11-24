package serving

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/governance"
	"github.com/kubeflow/semantic-operator/internal/planner"
)

func aiContext(t *testing.T, raw string) v1alpha1.AIContext {
	t.Helper()
	return v1alpha1.DecodeAIContext(&apiextensionsv1.JSON{Raw: []byte(raw)})
}

// aiContextModel is governedModel with ai_context added to the model, both
// metrics, and both dimensions. s_state uses the plain string form and the
// rest use the object form. The analyst can't see payroll_cost or
// manager_ssn, so their text is marked SECRET to catch any leak.
func aiContextModel(t *testing.T) *planner.CompiledModel {
	m := governedModel()
	m.AIContext = aiContext(t, `{"instructions":"Fiscal year starts in February.","examples":["Revenue by state last quarter"]}`)
	m.Metrics["revenue"].AIContext = aiContext(t, `{
		"instructions": "Use only for current-state analysis",
		"synonyms": ["sales", "gross revenue"],
		"examples": ["What is our revenue?", "Show revenue by state"]
	}`)
	m.Metrics["payroll_cost"].AIContext = aiContext(t, `{"instructions":"SECRET payroll guidance","examples":["SECRET payroll example"]}`)
	m.Datasets["store"].Fields["s_state"].AIContext = aiContext(t, `"Two-letter US state code."`)
	m.Datasets["store"].Fields["manager_ssn"].AIContext = aiContext(t, `{"instructions":"SECRET ssn guidance","synonyms":["SECRET ssn"]}`)
	return m
}

func TestMetricDiscoveryCarriesAIContext(t *testing.T) {
	svc := &Service{Store: NewStore()}
	metrics, err := svc.ListMetrics(aiContextModel(t), governance.Single("analyst"))
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("analyst should see only revenue, got %+v", metrics)
	}
	want := MetricInfo{
		Name:         "revenue",
		Instructions: "Use only for current-state analysis",
		Synonyms:     []string{"sales", "gross revenue"},
		Examples:     []string{"What is our revenue?", "Show revenue by state"},
	}
	if !reflect.DeepEqual(metrics[0], want) {
		t.Fatalf("ListMetrics:\n got %+v\nwant %+v", metrics[0], want)
	}
}

// When ai_context is a plain string, it is treated as the instructions.
func TestDimensionDiscoveryCarriesPlainStringAIContext(t *testing.T) {
	svc := &Service{Store: NewStore()}
	dims, err := svc.ListDimensions(aiContextModel(t), governance.Single("analyst"))
	if err != nil {
		t.Fatalf("ListDimensions: %v", err)
	}
	if len(dims) != 1 || dims[0].Name != "store.s_state" {
		t.Fatalf("analyst should see only store.s_state, got %+v", dims)
	}
	if dims[0].Instructions != "Two-letter US state code." {
		t.Fatalf("plain-string ai_context should become instructions, got %+v", dims[0])
	}
}

func TestDimensionDiscoveryCarriesStructuredAIContext(t *testing.T) {
	svc := &Service{Store: NewStore()}
	dims, err := svc.ListDimensions(aiContextModel(t), governance.Single("admin"))
	if err != nil {
		t.Fatalf("ListDimensions: %v", err)
	}
	if len(dims) != 2 || dims[1].Name != "store.manager_ssn" {
		t.Fatalf("admin should see both declared dimensions, got %+v", dims)
	}
	if dims[1].Instructions != "SECRET ssn guidance" || !reflect.DeepEqual(dims[1].Synonyms, []string{"SECRET ssn"}) {
		t.Fatalf("structured ai_context missing from dimension, got %+v", dims[1])
	}
}

// If a role can't see a metric or field, it must not see its ai_context
// either, in any listing.
func TestDiscoveryHidesAIContextOfDeniedElements(t *testing.T) {
	store := NewStore()
	m := aiContextModel(t)
	blob, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: store}
	id := governance.Single("analyst")

	metrics, err := svc.ListMetrics(m, id)
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	dims, err := svc.ListDimensions(m, id)
	if err != nil {
		t.Fatalf("ListDimensions: %v", err)
	}
	out, err := json.Marshal(map[string]any{"models": svc.Models(id), "metrics": metrics, "dimensions": dims})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "SECRET") {
		t.Fatalf("discovery leaked AI context of a denied element: %s", out)
	}
}

func TestModelsCarryModelLevelAIContext(t *testing.T) {
	store := NewStore()
	blob, err := json.Marshal(aiContextModel(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: store}

	models := svc.Models(governance.Single("analyst"))
	if len(models) != 1 {
		t.Fatalf("analyst should see the model, got %+v", models)
	}
	if models[0].Instructions != "Fiscal year starts in February." ||
		!reflect.DeepEqual(models[0].Examples, []string{"Revenue by state last quarter"}) {
		t.Fatalf("model-level ai_context missing, got %+v", models[0])
	}

	if got := svc.Models(governance.Single("stranger")); len(got) != 0 {
		t.Fatalf("a role with no policy must not see the model or its instructions, got %+v", got)
	}
}

// A model without ai_context must produce the same JSON as before, with no
// new keys, so existing clients keep working.
func TestEmptyAIContextIsOmitted(t *testing.T) {
	store := NewStore()
	m := governedModel()
	blob, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("sm-retail-compiled", blob); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: store}
	id := governance.Single("admin")

	metrics, err := svc.ListMetrics(m, id)
	if err != nil {
		t.Fatalf("ListMetrics: %v", err)
	}
	dims, err := svc.ListDimensions(m, id)
	if err != nil {
		t.Fatalf("ListDimensions: %v", err)
	}
	for _, v := range []any{svc.Models(id), metrics, dims} {
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{`"instructions"`, `"synonyms"`, `"examples"`} {
			if strings.Contains(string(out), key) {
				t.Fatalf("empty ai_context should omit %s, got %s", key, out)
			}
		}
	}
}
