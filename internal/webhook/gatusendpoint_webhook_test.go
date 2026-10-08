package webhook

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/Wihrt/gatus-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func makeEndpoint(conditions []string) *monitoringv1alpha1.GatusEndpoint {
	return &monitoringv1alpha1.GatusEndpoint{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ep", Namespace: "default"},
		Spec: monitoringv1alpha1.GatusEndpointSpec{
			Name:       "Test EP",
			URL:        "https://example.com",
			Conditions: conditions,
		},
	}
}

func TestEndpointWebhook_AllowsValidStatusCondition(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[STATUS] == 200"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected valid condition to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_AllowsBodyJSONPath(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[BODY].user.name == john"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected valid body condition to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_AllowsLenFunction(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"len([BODY].data) < 5"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected len([BODY]) condition to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_AllowsResponseTime(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[RESPONSE_TIME] < 500"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected [RESPONSE_TIME] condition to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_AllowsCertificateExpiration(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[CERTIFICATE_EXPIRATION] > 48h"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected [CERTIFICATE_EXPIRATION] condition to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_RejectsUnknownPlaceholder(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[FOOBAR] == 200"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err == nil {
		t.Error("expected unknown placeholder to be rejected")
	}
}

func TestEndpointWebhook_RejectsMissingOperator(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[STATUS] 200"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err == nil {
		t.Error("expected missing operator to be rejected")
	}
}

func TestEndpointWebhook_RejectsEmptyCondition(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{""})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err == nil {
		t.Error("expected empty condition to be rejected")
	}
}

func TestEndpointWebhook_AllowsNoConditions(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint(nil)
	_, err := v.ValidateCreate(context.Background(), ep)
	if err != nil {
		t.Errorf("expected endpoint with no conditions to be accepted, got: %v", err)
	}
}

func TestEndpointWebhook_ValidateUpdate(t *testing.T) {
	v := &GatusEndpointValidator{}
	old := makeEndpoint([]string{"[STATUS] == 200"})
	updated := makeEndpoint([]string{"[FOOBAR] == 500"})
	_, err := v.ValidateUpdate(context.Background(), old, updated)
	if err == nil {
		t.Error("expected invalid updated condition to be rejected")
	}
}

func TestEndpointWebhook_ValidateDelete(t *testing.T) {
	v := &GatusEndpointValidator{}
	ep := makeEndpoint([]string{"[STATUS] == 200"})
	_, err := v.ValidateDelete(context.Background(), ep)
	if err != nil {
		t.Errorf("ValidateDelete should always succeed, got: %v", err)
	}
}

func TestValidateCondition_LenWithNonBody(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	// len() works with any valid placeholder per Gatus source.
	err := validateCondition("len([STATUS]) < 5", fld)
	if err != nil {
		t.Errorf("expected len() with valid placeholder to be accepted, got: %v", err)
	}
}

func TestValidateCondition_LenWithNonBodyInCompoundExpression(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	// len() works with any valid placeholder per Gatus source.
	err := validateCondition("[STATUS] == 200 && len([STATUS]) < 5", fld)
	if err != nil {
		t.Errorf("expected len() with valid placeholder in compound expression to be accepted, got: %v", err)
	}
}

func TestValidateCondition_HasWithNonBodyInCompoundExpression(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	// has() works with any valid placeholder per Gatus source.
	err := validateCondition("[STATUS] == 200 && has([STATUS])", fld)
	if err != nil {
		t.Errorf("expected has() with valid placeholder in compound expression to be accepted, got: %v", err)
	}
}

func TestValidateCondition_LenWithInvalidPlaceholder(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	err := validateCondition("len([FOOBAR]) < 5", fld)
	if err == nil {
		t.Error("expected len() with unknown placeholder to be rejected")
	}
}

func TestValidateCondition_HasWithInvalidPlaceholder(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	err := validateCondition("has([FOOBAR]) == true", fld)
	if err == nil {
		t.Error("expected has() with unknown placeholder to be rejected")
	}
}

func TestValidateCondition_ContextPlaceholder(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	err := validateCondition("[CONTEXT].user_id == 123", fld)
	if err != nil {
		t.Errorf("expected [CONTEXT] placeholder to be accepted, got: %v", err)
	}
}

func TestValidateCondition_LenWithContext(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	err := validateCondition("len([CONTEXT].items) > 0", fld)
	if err != nil {
		t.Errorf("expected len() with [CONTEXT] placeholder to be accepted, got: %v", err)
	}
}

// Operators are only recognised by Gatus when surrounded by single spaces
// (config/endpoint/condition.go in Gatus v5.37.0).
func TestValidateCondition_Operators(t *testing.T) {
	fld := field.NewPath("spec").Child("conditions").Index(0)
	tests := []struct {
		name      string
		condition string
		wantErr   bool
	}{
		{"eq", "[STATUS] == 200", false},
		{"neq", "[STATUS] != 500", false},
		{"lte", "[RESPONSE_TIME] <= 500", false},
		{"gte", "[CERTIFICATE_EXPIRATION] >= 48h", false},
		{"gt", "[CERTIFICATE_EXPIRATION] > 48h", false},
		{"lt", "[RESPONSE_TIME] < 500", false},
		{"len with lt", "len([BODY].items) < 5", false},
		{"pat function", "[IP] == pat(192.168.*.*)", false},
		{"any function", "[IP] == any(1.1.1.1, 1.0.0.1)", false},
		{"has function", "has([BODY].errors) == false", false},
		{"lowercase placeholder", "[status] == 200", false},
		{"lowercase placeholder in len", "len([body].items) > 0", false},
		{"no spaces eq", "[STATUS]==200", true},
		{"no spaces neq", "[STATUS]!=200", true},
		{"no spaces lte", "[RESPONSE_TIME]<=500", true},
		{"no spaces gte", "[RESPONSE_TIME]>=500", true},
		{"no spaces lt", "[RESPONSE_TIME]<500", true},
		{"no spaces gt", "[RESPONSE_TIME]>500", true},
		{"space only before", "[STATUS] ==200", true},
		{"space only after", "[STATUS]== 200", true},
		{"trailing operator without right space", "[STATUS] ==", true},
		{"no operator", "[STATUS] 200", true},
		{"unknown placeholder", "[UNKNOWN_PLACEHOLDER] == 200", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCondition(tt.condition, fld)
			if tt.wantErr && err == nil {
				t.Errorf("expected %q to be rejected", tt.condition)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected %q to be accepted, got: %v", tt.condition, err)
			}
		})
	}
}

func TestEndpointWebhook_RejectsMixedValidAndUnknownPlaceholder(t *testing.T) {
	v := &GatusEndpointValidator{}
	// [STATUS] is valid but [FOO] is not; the condition should be rejected.
	ep := makeEndpoint([]string{"[STATUS] == 200 && [FOO] != 0"})
	_, err := v.ValidateCreate(context.Background(), ep)
	if err == nil {
		t.Error("expected condition with unknown placeholder [FOO] to be rejected even when a valid placeholder is also present")
	}
}
