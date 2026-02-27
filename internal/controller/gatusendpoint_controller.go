package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"gopkg.in/yaml.v3"

	monitoringv1alpha1 "github.com/Wihrt/gatus-ingress-controller/api/v1alpha1"
)

const (
	configMapName = "gatus-config"
	configMapKey  = "endpoints.yaml"
)

type GatusEndpointReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	TargetNamespace string
}

type gatusEndpointConfig struct {
	Name       string             `yaml:"name"`
	Group      string             `yaml:"group,omitempty"`
	URL        string             `yaml:"url"`
	Conditions []string           `yaml:"conditions,omitempty"`
	Alerts     []gatusAlertConfig `yaml:"alerts,omitempty"`
	DNS        *gatusDNSConfig    `yaml:"dns,omitempty"`
	UI         *gatusUIConfig     `yaml:"ui,omitempty"`
}

type gatusAlertConfig struct {
	Type             string `yaml:"type"`
	FailureThreshold int    `yaml:"failure-threshold"`
	SendOnResolved   bool   `yaml:"send-on-resolved"`
}

type gatusDNSConfig struct {
	QueryName string `yaml:"query-name"`
	QueryType string `yaml:"query-type"`
}

type gatusUIConfig struct {
	HideHostname bool `yaml:"hide-hostname,omitempty"`
	HideURL      bool `yaml:"hide-url,omitempty"`
}

type gatusConfig struct {
	Endpoints []gatusEndpointConfig `yaml:"endpoints"`
}

func (r *GatusEndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling GatusEndpoint", "name", req.Name, "namespace", req.Namespace)

	endpointList := &monitoringv1alpha1.GatusEndpointList{}
	if err := r.List(ctx, endpointList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list GatusEndpoints: %w", err)
	}

	var endpoints []gatusEndpointConfig
	for _, ep := range endpointList.Items {
		var alertConfigs []gatusAlertConfig
		for _, alertRef := range ep.Spec.Alerts {
			ns := alertRef.Namespace
			if ns == "" {
				ns = ep.Namespace
			}
			alert := &monitoringv1alpha1.GatusAlert{}
			if err := r.Get(ctx, types.NamespacedName{Name: alertRef.Name, Namespace: ns}, alert); err != nil {
				logger.Error(err, "Failed to get GatusAlert", "name", alertRef.Name, "namespace", ns)
				continue
			}
			alertConfigs = append(alertConfigs, gatusAlertConfig{
				Type:             alert.Spec.Type,
				FailureThreshold: alert.Spec.FailureThreshold,
				SendOnResolved:   alert.Spec.SendOnResolved,
			})
		}

		epConfig := gatusEndpointConfig{
			Name:       ep.Spec.Name,
			Group:      ep.Spec.Group,
			URL:        ep.Spec.URL,
			Conditions: ep.Spec.Conditions,
			Alerts:     alertConfigs,
		}

		if ep.Spec.DNS != nil {
			epConfig.DNS = &gatusDNSConfig{
				QueryName: ep.Spec.DNS.QueryName,
				QueryType: ep.Spec.DNS.QueryType,
			}
		}

		if ep.Spec.UI != nil {
			epConfig.UI = &gatusUIConfig{
				HideHostname: ep.Spec.UI.HideHostname,
				HideURL:      ep.Spec.UI.HideURL,
			}
		}

		endpoints = append(endpoints, epConfig)
	}

	cfg := gatusConfig{Endpoints: endpoints}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to marshal Gatus config: %w", err)
	}

	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: r.TargetNamespace}, cm)
	if errors.IsNotFound(err) {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: r.TargetNamespace,
			},
			Data: map[string]string{
				configMapKey: string(data),
			},
		}
		if err := r.Create(ctx, cm); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create ConfigMap: %w", err)
		}
		logger.Info("Created ConfigMap", "name", configMapName, "namespace", r.TargetNamespace)
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get ConfigMap: %w", err)
	} else {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[configMapKey] = string(data)
		if err := r.Update(ctx, cm); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update ConfigMap: %w", err)
		}
		logger.Info("Updated ConfigMap", "name", configMapName, "namespace", r.TargetNamespace)
	}

	return ctrl.Result{}, nil
}

func (r *GatusEndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.GatusEndpoint{}).
		Complete(r)
}
