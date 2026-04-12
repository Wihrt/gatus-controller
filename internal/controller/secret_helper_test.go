package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpsertConfigMapKey_CreatesKeyInExistingConfigMap(t *testing.T) {
	s := newTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm).Build()

	result, err := upsertConfigMapKey(context.Background(), fakeClient, "test-ns", "test-cm", "my-key", "my-value")
	if err != nil {
		t.Fatalf("upsertConfigMapKey returned unexpected error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got RequeueAfter=%v", result.RequeueAfter)
	}

	updated := &corev1.ConfigMap{}
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, updated); err != nil {
		t.Fatalf("failed to get configmap: %v", err)
	}
	if updated.Data["my-key"] != "my-value" {
		t.Errorf("expected 'my-value', got %q", updated.Data["my-key"])
	}
}

func TestUpsertConfigMapKey_UpdatesExistingKey(t *testing.T) {
	s := newTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
		Data:       map[string]string{"my-key": "old-value"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm).Build()

	_, err := upsertConfigMapKey(context.Background(), fakeClient, "test-ns", "test-cm", "my-key", "new-value")
	if err != nil {
		t.Fatalf("upsertConfigMapKey returned unexpected error: %v", err)
	}

	updated := &corev1.ConfigMap{}
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, updated); err != nil {
		t.Fatalf("failed to get configmap: %v", err)
	}
	if updated.Data["my-key"] != "new-value" {
		t.Errorf("expected 'new-value', got %q", updated.Data["my-key"])
	}
}

func TestUpsertConfigMapKey_PreservesOtherKeys(t *testing.T) {
	s := newTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
		Data:       map[string]string{"existing-key": "existing-value"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm).Build()

	_, err := upsertConfigMapKey(context.Background(), fakeClient, "test-ns", "test-cm", "new-key", "new-value")
	if err != nil {
		t.Fatalf("upsertConfigMapKey returned unexpected error: %v", err)
	}

	updated := &corev1.ConfigMap{}
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, updated); err != nil {
		t.Fatalf("failed to get configmap: %v", err)
	}
	if updated.Data["existing-key"] != "existing-value" {
		t.Errorf("existing key was modified: got %q", updated.Data["existing-key"])
	}
	if updated.Data["new-key"] != "new-value" {
		t.Errorf("new key not set: got %q", updated.Data["new-key"])
	}
}

func TestUpsertConfigMapKey_RequeuesWhenConfigMapNotFound(t *testing.T) {
	s := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	result, err := upsertConfigMapKey(context.Background(), fakeClient, "test-ns", "missing-cm", "key", "value")
	if err != nil {
		t.Fatalf("upsertConfigMapKey should not return error for missing ConfigMap, got: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected RequeueAfter > 0 when ConfigMap is missing")
	}
}

func TestUpsertConfigMapKey_HandlesNilDataMap(t *testing.T) {
	s := newTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
		// Data is nil
	}
	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(cm).Build()

	_, err := upsertConfigMapKey(context.Background(), fakeClient, "test-ns", "test-cm", "key", "value")
	if err != nil {
		t.Fatalf("upsertConfigMapKey should handle nil Data map, got: %v", err)
	}

	updated := &corev1.ConfigMap{}
	_ = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, updated)
	if updated.Data["key"] != "value" {
		t.Errorf("expected 'value', got %q", updated.Data["key"])
	}
}
