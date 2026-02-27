package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatusAlertSpec struct {
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// +optional
	WebhookURL string `json:"webhookUrl,omitempty"`
	// +kubebuilder:default=5
	FailureThreshold int `json:"failureThreshold"`
	// +kubebuilder:default=true
	SendOnResolved bool `json:"sendOnResolved"`
}

type GatusAlertStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ga
type GatusAlert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatusAlertSpec   `json:"spec,omitempty"`
	Status GatusAlertStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type GatusAlertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatusAlert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatusAlert{}, &GatusAlertList{})
}
