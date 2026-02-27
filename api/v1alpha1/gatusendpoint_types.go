package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatusAlertRef struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type GatusDNSConfig struct {
	// +kubebuilder:validation:Required
	QueryName string `json:"queryName"`
	// +kubebuilder:default=A
	QueryType string `json:"queryType"`
}

type GatusUIConfig struct {
	// +optional
	HideHostname bool `json:"hideHostname,omitempty"`
	// +optional
	HideURL bool `json:"hideUrl,omitempty"`
}

type GatusEndpointSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +optional
	Group string `json:"group,omitempty"`
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// +optional
	Conditions []string `json:"conditions,omitempty"`
	// +optional
	Alerts []GatusAlertRef `json:"alerts,omitempty"`
	// +optional
	DNS *GatusDNSConfig `json:"dns,omitempty"`
	// +optional
	UI *GatusUIConfig `json:"ui,omitempty"`
}

type GatusEndpointStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ge
type GatusEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatusEndpointSpec   `json:"spec,omitempty"`
	Status GatusEndpointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type GatusEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatusEndpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatusEndpoint{}, &GatusEndpointList{})
}
