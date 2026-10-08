package controller

import (
	"context"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitoringv1alpha1 "github.com/Wihrt/gatus-controller/api/v1alpha1"
)

const externalEndpointsKey = "external-endpoints.yaml"

// GatusExternalEndpointReconciler reconciles GatusExternalEndpoint resources and
// aggregates them into the gatus ConfigMap under the external-endpoints.yaml key.
type GatusExternalEndpointReconciler struct {
	client.Client
	TargetNamespace string
	ConfigMapName   string
}

// --- Internal YAML representation for external endpoints ---

type gatusExternalConfigFile struct {
	ExternalEndpoints []gatusExternalEndpointYAML `yaml:"external-endpoints"`
}

type gatusExternalEndpointYAML struct {
	Name      string              `yaml:"name"`
	Enabled   *bool               `yaml:"enabled,omitempty"`
	Group     string              `yaml:"group,omitempty"`
	Token     string              `yaml:"token"`
	Alerts    []gatusAlertYAML    `yaml:"alerts,omitempty"`
	Heartbeat *gatusHeartbeatYAML `yaml:"heartbeat,omitempty"`
}

type gatusHeartbeatYAML struct {
	Interval string `yaml:"interval,omitempty"`
}

// Reconcile aggregates all GatusExternalEndpoints and writes external-endpoints.yaml
// to the shared gatus Secret.
func (r *GatusExternalEndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling GatusExternalEndpoint", "name", req.Name, "namespace", req.Namespace)

	// List all GatusExternalEndpoints across all namespaces.
	extList := &monitoringv1alpha1.GatusExternalEndpointList{}
	if err := r.List(ctx, extList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list GatusExternalEndpoints: %w", err)
	}

	// Gatus also rejects an external endpoint whose key equals a regular
	// endpoint's key; regular endpoints win. Which regular CR owns a key
	// is irrelevant here, so the first one in sorted order is only reported.
	endpointList := &monitoringv1alpha1.GatusEndpointList{}
	if err := r.List(ctx, endpointList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list GatusEndpoints: %w", err)
	}
	sort.Slice(endpointList.Items, func(i, j int) bool {
		ki := endpointList.Items[i].Namespace + "/" + endpointList.Items[i].Name
		kj := endpointList.Items[j].Namespace + "/" + endpointList.Items[j].Name
		return ki < kj
	})
	endpointWinners := make(map[string]string, len(endpointList.Items))
	for _, ep := range endpointList.Items {
		key := gatusEndpointKey(ep.Spec.Group, ep.Spec.Name)
		if _, ok := endpointWinners[key]; !ok {
			endpointWinners[key] = ep.Namespace + "/" + ep.Name
		}
	}

	// Sort for deterministic output.
	sort.Slice(extList.Items, func(i, j int) bool {
		ki := extList.Items[i].Namespace + "/" + extList.Items[i].Name
		kj := extList.Items[j].Namespace + "/" + extList.Items[j].Name
		return ki < kj
	})

	// Keep only the first CR per Gatus key (group + name, sanitised).
	var externalEndpoints []gatusExternalEndpointYAML
	winners := make(map[string]string, len(extList.Items))
	for _, ext := range extList.Items {
		id := ext.Namespace + "/" + ext.Name
		key := gatusEndpointKey(ext.Spec.Group, ext.Spec.Name)
		if winner, dup := endpointWinners[key]; dup {
			logger.Info("Skipping GatusExternalEndpoint whose Gatus key is used by a GatusEndpoint",
				"externalEndpoint", id, "key", key, "winner", winner)
			continue
		}
		if winner, dup := winners[key]; dup {
			logger.Info("Skipping GatusExternalEndpoint with duplicate Gatus key",
				"externalEndpoint", id, "key", key, "winner", winner)
			continue
		}
		winners[key] = id
		alertYAMLs := convertAlerts(ext.Spec.Alerts)

		extYAML := gatusExternalEndpointYAML{
			Name:    ext.Spec.Name,
			Enabled: boolPtr(ext.Spec.Enabled),
			Group:   ext.Spec.Group,
			Token:   ext.Spec.Token,
			Alerts:  alertYAMLs,
		}

		if ext.Spec.Heartbeat != nil && ext.Spec.Heartbeat.Interval != "" {
			extYAML.Heartbeat = &gatusHeartbeatYAML{
				Interval: ext.Spec.Heartbeat.Interval,
			}
		}

		externalEndpoints = append(externalEndpoints, extYAML)
	}

	cfg := gatusExternalConfigFile{ExternalEndpoints: externalEndpoints}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to marshal Gatus external endpoints config: %w", err)
	}

	return upsertConfigMapKey(ctx, r.Client, r.TargetNamespace, r.ConfigMapName, externalEndpointsKey, string(data))
}

func (r *GatusExternalEndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.GatusExternalEndpoint{}).
		// Regular endpoints take precedence on key conflicts, so any change to
		// them can change the output. Every reconcile re-lists everything, hence
		// a single fixed request is enough.
		Watches(&monitoringv1alpha1.GatusEndpoint{}, handler.EnqueueRequestsFromMapFunc(
			func(context.Context, client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: "gatusendpoint-change"}}}
			})).
		Complete(r)
}
