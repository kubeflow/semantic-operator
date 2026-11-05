package controllers

// TestStatusConditionsAfterFirstReconcile_ExposesTransientLoss demonstrates that
// after a single reconcile pass on a new SemanticModel (where reconcileViews triggers
// r.Update to persist the owned-views annotation), the Validated / Compiled / Published
// conditions are wiped from in-memory sm by the fake client copying the old stored
// status back into the object pointer.  Only ViewsReady survives because it is set
// AFTER reconcileViews returns.
//
// The existing TestReconcileHappyPath masks this because it calls Reconcile twice
// (reconcileOnce runs a loop of 2). On the second pass the annotation is already
// set, so r.Update inside reconcileViews is skipped and all conditions are written
// cleanly.  Monitoring or admission webhooks that read status between pass-1 and
// pass-2 (up to 5 minutes, the default ResyncPeriod) will see missing conditions.
//
// If this test FAILS (conditions missing) → bug confirmed.
// If this test PASSES (all four True) → the fake client behaviour changed or the
// ordering was fixed.

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	semanticv1alpha1 "github.com/kubeflow/semantic-operator/api/v1alpha1"
	"github.com/kubeflow/semantic-operator/internal/emitter"
)

func TestStatusConditionsAfterFirstReconcile(t *testing.T) {
	scheme := testScheme(t)
	cr := demoCR() // SemanticModel with one view declared
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).
		WithStatusSubresource(&semanticv1alpha1.SemanticModel{}).Build()
	srx := &fakeStarRocks{tables: healthyTables()}
	d, _ := emitter.Get("starrocks")
	r := &SemanticModelReconciler{Client: cl, DB: srx, Dialect: d}

	name := types.NamespacedName{Name: "retail", Namespace: "default"}

	// === Pass 1 only — do NOT call reconcileOnce (which runs twice) ===
	// Pass 1: adds finalizer then immediately continues through the full reconcile
	// (no early return in the success path after finalizer add).
	// reconcileViews() will call r.Update(ctx, sm) because the annotation changes
	// from "" → json([view]).  The fake client then copies the old stored status
	// (still empty at this point) back into sm, wiping Validated/Compiled/Published.
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: name}); err != nil {
		t.Fatalf("first reconcile returned error: %v", err)
	}

	var got semanticv1alpha1.SemanticModel
	if err := cl.Get(context.Background(), name, &got); err != nil {
		t.Fatal(err)
	}

	t.Logf("conditions after first reconcile: %+v", got.Status.Conditions)

	missing := false
	for _, cond := range []string{
		semanticv1alpha1.ConditionValidated,
		semanticv1alpha1.ConditionCompiled,
		semanticv1alpha1.ConditionPublished,
		semanticv1alpha1.ConditionViewsReady,
	} {
		if !meta.IsStatusConditionTrue(got.Status.Conditions, cond) {
			t.Errorf("BUG: condition %s is not True after first reconcile (got %+v)", cond, got.Status.Conditions)
			missing = true
		}
	}
	if missing {
		t.Log("CONCLUSION: r.Update() inside reconcileViews() wipes in-memory conditions set")
		t.Log("earlier in the same reconcile pass.  Conditions are incomplete until the")
		t.Log("second reconcile fires (default 5-minute ResyncPeriod).")
	} else {
		t.Log("All conditions set after first reconcile — bug may have been fixed or fake client behaviour changed.")
	}
}

// TestStatusConditionsAfterSecondReconcile verifies the conditions are fully set
// after two reconcile passes (the current TestReconcileHappyPath path).
// This should always pass — it confirms the second pass fixes the transient gap.
func TestStatusConditionsAfterSecondReconcile(t *testing.T) {
	scheme := testScheme(t)
	cr := demoCR()
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).
		WithStatusSubresource(&semanticv1alpha1.SemanticModel{}).Build()
	srx := &fakeStarRocks{tables: healthyTables()}
	d, _ := emitter.Get("starrocks")
	r := &SemanticModelReconciler{Client: cl, DB: srx, Dialect: d}

	name := types.NamespacedName{Name: "retail", Namespace: "default"}
	reconcileOnce(t, r, name) // runs twice

	var got semanticv1alpha1.SemanticModel
	if err := cl.Get(context.Background(), name, &got); err != nil {
		t.Fatal(err)
	}
	for _, cond := range []string{
		semanticv1alpha1.ConditionValidated,
		semanticv1alpha1.ConditionCompiled,
		semanticv1alpha1.ConditionPublished,
		semanticv1alpha1.ConditionViewsReady,
	} {
		if !meta.IsStatusConditionTrue(got.Status.Conditions, cond) {
			t.Errorf("condition %s not true after second reconcile: %+v", cond, got.Status.Conditions)
		}
	}
}
