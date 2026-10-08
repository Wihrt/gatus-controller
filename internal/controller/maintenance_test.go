package controller

import (
	"context"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	monitoringv1alpha1 "github.com/Wihrt/gatus-controller/api/v1alpha1"
)

func TestMaintenanceDays(t *testing.T) {
	tests := []struct {
		name  string
		day   string
		every []string
		want  []string
	}{
		{"lowercase", "", []string{"monday", "thursday"}, []string{"Monday", "Thursday"}},
		{"mixed case and spaces", "", []string{" mOnDaY ", "FRIDAY"}, []string{"Monday", "Friday"}},
		{"day merged first", "friday", []string{"monday"}, []string{"Friday", "Monday"}},
		{"duplicates removed", "monday", []string{"Monday", "tuesday", "TUESDAY"}, []string{"Monday", "Tuesday"}},
		{"empty", "", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maintenanceDays(tt.day, tt.every); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("maintenanceDays(%q, %v) = %v, want %v", tt.day, tt.every, got, tt.want)
			}
		})
	}
}

// TestGatusEndpointReconciler_MaintenanceWindowDays verifies that the deprecated day
// field is merged into every (no day: key emitted) and that enabled is emitted only when set.
func TestGatusEndpointReconciler_MaintenanceWindowDays(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)
	disabled := false

	ep := &monitoringv1alpha1.GatusEndpoint{
		ObjectMeta: metav1.ObjectMeta{Name: "mw-days", Namespace: "default"},
		Spec: monitoringv1alpha1.GatusEndpointSpec{
			Name: "MW Days",
			URL:  "https://example.com",
			MaintenanceWindows: []monitoringv1alpha1.GatusMaintenanceWindow{
				{
					Day:      "friday",
					Every:    []string{"monday", "thursday", "Friday", "monday"},
					Start:    "23:00",
					Duration: "1h",
					Enabled:  &disabled,
				},
			},
		},
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "gatus-config", Namespace: "gatus"}}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm, ep).Build()
	r := &GatusEndpointReconciler{Client: fakeClient, Scheme: s, TargetNamespace: "gatus", ConfigMapName: "gatus-config"}

	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "mw-days", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	updated := &corev1.ConfigMap{}
	_ = fakeClient.Get(ctx, types.NamespacedName{Name: "gatus-config", Namespace: "gatus"}, updated)
	y := updated.Data["endpoints.yaml"]

	if strings.Contains(y, "day:") {
		t.Errorf("unexpected day: key in output:\n%s", y)
	}
	if !strings.Contains(y, "- Friday\n") || !strings.Contains(y, "- Monday\n") || !strings.Contains(y, "- Thursday\n") {
		t.Errorf("expected capitalised days in output, got:\n%s", y)
	}
	if strings.Count(y, "Monday") != 1 || strings.Count(y, "Friday") != 1 {
		t.Errorf("expected de-duplicated days, got:\n%s", y)
	}
	if strings.Index(y, "Friday") > strings.Index(y, "Monday") {
		t.Errorf("expected day merged before every entries, got:\n%s", y)
	}
	if !strings.Contains(y, "enabled: false") {
		t.Errorf("expected enabled: false in output, got:\n%s", y)
	}
}

// TestGatusEndpointReconciler_MaintenanceWindowEnabledOmitted verifies enabled is not emitted when unset.
func TestGatusEndpointReconciler_MaintenanceWindowEnabledOmitted(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)

	ep := &monitoringv1alpha1.GatusEndpoint{
		ObjectMeta: metav1.ObjectMeta{Name: "mw-en", Namespace: "default"},
		Spec: monitoringv1alpha1.GatusEndpointSpec{
			Name: "MW En",
			URL:  "https://example.com",
			MaintenanceWindows: []monitoringv1alpha1.GatusMaintenanceWindow{
				{Every: []string{"sunday"}, Start: "01:00", Duration: "30m"},
			},
		},
	}
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "gatus-config", Namespace: "gatus"}}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm, ep).Build()
	r := &GatusEndpointReconciler{Client: fakeClient, Scheme: s, TargetNamespace: "gatus", ConfigMapName: "gatus-config"}

	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "mw-en", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	updated := &corev1.ConfigMap{}
	_ = fakeClient.Get(ctx, types.NamespacedName{Name: "gatus-config", Namespace: "gatus"}, updated)
	y := updated.Data["endpoints.yaml"]
	if !strings.Contains(y, "- Sunday") {
		t.Errorf("expected Sunday in output, got:\n%s", y)
	}
	// the endpoint-level enabled key may appear; the window must not carry its own.
	if i := strings.Index(y, "maintenance-windows:"); i < 0 || strings.Contains(y[i:], "enabled:") {
		t.Errorf("enabled must not be emitted in the window when unset, got:\n%s", y)
	}
}
