package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatusAlertSpec struct {
	// Type is the alert provider type (e.g., "discord", "slack", "pagerduty").
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Enabled indicates whether this alert is enabled globally.
	// +kubebuilder:default=true
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// WebhookURL is the webhook URL for providers that support it (e.g. discord, slack).
	// +optional
	WebhookURL string `json:"webhookUrl,omitempty"`

	// FailureThreshold is the default number of consecutive failures before triggering the alert.
	// Can be overridden at the endpoint level via GatusAlertRef.
	// +kubebuilder:default=3
	FailureThreshold int `json:"failureThreshold"`

	// SuccessThreshold is the default number of consecutive successes before resolving an ongoing incident.
	// Can be overridden at the endpoint level via GatusAlertRef.
	// +kubebuilder:default=2
	SuccessThreshold int `json:"successThreshold"`

	// SendOnResolved indicates whether to send a notification once a triggered alert is resolved.
	// Can be overridden at the endpoint level via GatusAlertRef.
	// +optional
	SendOnResolved bool `json:"sendOnResolved,omitempty"`

	// Description is the default description included in the alert notification.
	// Can be overridden at the endpoint level via GatusAlertRef.
	// +optional
	Description string `json:"description,omitempty"`

	// MinimumReminderInterval is the minimum duration between alert reminders (e.g. "30m", "1h").
	// Set to "0" or leave empty to disable reminders.
	// +optional
	MinimumReminderInterval string `json:"minimumReminderInterval,omitempty"`
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
