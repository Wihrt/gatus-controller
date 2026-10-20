package controller

import (
	"context"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	monitoringv1alpha1 "github.com/Wihrt/gatus-controller/api/v1alpha1"
)

func dedupEndpoint(ns, name, group, specName, url string) *monitoringv1alpha1.GatusEndpoint {
	return &monitoringv1alpha1.GatusEndpoint{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       monitoringv1alpha1.GatusEndpointSpec{Name: specName, Group: group, URL: url},
	}
}

func dedupExternal(ns, name, group, specName string) *monitoringv1alpha1.GatusExternalEndpoint {
	return &monitoringv1alpha1.GatusExternalEndpoint{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       monitoringv1alpha1.GatusExternalEndpointSpec{Name: specName, Group: group, Token: "t-" + ns},
	}
}

func reqFor(name, namespace string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}}
}

// emittedEntries returns "group/name" of every entry written under the given
// ConfigMap key / top-level YAML field, in output order.
func emittedEntries(t *testing.T, c client.Client, cmKey, field string) []string {
	t.Helper()
	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "gatus-config", Namespace: "gatus"}, cm); err != nil {
		t.Fatalf("ConfigMap not found: %v", err)
	}
	var out map[string][]map[string]interface{}
	if err := yaml.Unmarshal([]byte(cm.Data[cmKey]), &out); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}
	var res []string
	for _, e := range out[field] {
		g, _ := e["group"].(string)
		n, _ := e["name"].(string)
		res = append(res, g+"/"+n)
	}
	return res
}

func reconcileEndpoints(t *testing.T, objs ...client.Object) []string {
	t.Helper()
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
		WithObjects(append(objs, extEndpointConfigMap())...).Build()
	r := &GatusEndpointReconciler{Client: c, TargetNamespace: "gatus", ConfigMapName: "gatus-config"}
	reconcileEndpointReq(t, r)
	return emittedEntries(t, c, "endpoints.yaml", "endpoints")
}

func reconcileExternals(t *testing.T, objs ...client.Object) []string {
	t.Helper()
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
		WithObjects(append(objs, extEndpointConfigMap())...).Build()
	r := newExtEndpointReconciler(c)
	reconcileExtEndpoint(t, r, "x", "x")
	return emittedEntries(t, c, "external-endpoints.yaml", "external-endpoints")
}

func reconcileEndpointReq(t *testing.T, r *GatusEndpointReconciler) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), reqFor("x", "x")); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
}

func TestGatusEndpointReconciler_DedupSameNameAndGroup(t *testing.T) {
	got := reconcileEndpoints(t,
		dedupEndpoint("zeta", "a", "core", "API", "https://zeta"),
		dedupEndpoint("alpha", "b", "core", "API", "https://alpha"),
	)
	if want := []string{"core/API"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// The winner must be alpha/b: check the URL.
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(
		dedupEndpoint("zeta", "a", "core", "API", "https://zeta"),
		dedupEndpoint("alpha", "b", "core", "API", "https://alpha"),
		extEndpointConfigMap()).Build()
	r := &GatusEndpointReconciler{Client: c, TargetNamespace: "gatus", ConfigMapName: "gatus-config"}
	reconcileEndpointReq(t, r)
	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "gatus-config", Namespace: "gatus"}, cm); err != nil {
		t.Fatal(err)
	}
	var out struct {
		Endpoints []struct {
			URL string `yaml:"url"`
		} `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal([]byte(cm.Data["endpoints.yaml"]), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Endpoints) != 1 || out.Endpoints[0].URL != "https://alpha" {
		t.Fatalf("expected alpha/b to win, got %+v", out.Endpoints)
	}
}

func TestGatusEndpointReconciler_DedupSameNameDifferentGroups(t *testing.T) {
	got := reconcileEndpoints(t,
		dedupEndpoint("ns1", "a", "prod", "API", "https://1"),
		dedupEndpoint("ns2", "a", "staging", "API", "https://2"),
	)
	if want := []string{"prod/API", "staging/API"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGatusEndpointReconciler_DedupSanitisedKeyCollision(t *testing.T) {
	tests := []struct {
		name         string
		g1, n1       string
		g2, n2       string
		wantSurvivor string
	}{
		{"case", "Core", "API", "core", "api", "Core/API"},
		{"underscore vs dash", "core", "my_api", "core", "my-api", "core/my_api"},
		{"space and dot", "core", "my api", "core", "my.api", "core/my api"},
		{"trim", "core", " api", "core", "api", "core/ api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconcileEndpoints(t,
				dedupEndpoint("a", "x", tt.g1, tt.n1, "https://1"),
				dedupEndpoint("b", "x", tt.g2, tt.n2, "https://2"),
			)
			if want := []string{tt.wantSurvivor}; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestGatusExternalEndpointReconciler_DedupSameNameAndGroup(t *testing.T) {
	got := reconcileExternals(t,
		dedupExternal("zeta", "a", "core", "Job"),
		dedupExternal("alpha", "a", "core", "Job"),
	)
	if want := []string{"core/Job"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGatusExternalEndpointReconciler_DedupDifferentGroups(t *testing.T) {
	got := reconcileExternals(t,
		dedupExternal("ns1", "a", "prod", "Job"),
		dedupExternal("ns2", "a", "staging", "Job"),
	)
	if want := []string{"prod/Job", "staging/Job"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGatusExternalEndpointReconciler_DedupSanitisedKeyCollision(t *testing.T) {
	got := reconcileExternals(t,
		dedupExternal("a", "x", "core", "my_job"),
		dedupExternal("b", "x", "Core", "My-Job"),
	)
	if want := []string{"core/my_job"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGatusExternalEndpointReconciler_DropsKeyUsedByGatusEndpoint(t *testing.T) {
	got := reconcileExternals(t,
		// Sorts before the regular endpoint's namespace, but regular endpoints win.
		dedupExternal("a", "x", "core", "API"),
		dedupExternal("a", "y", "core", "Other"),
		dedupEndpoint("z", "x", "Core", "api", "https://1"),
	)
	if want := []string{"core/Other"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
