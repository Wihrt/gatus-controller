package webhook

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	monitoringv1alpha1 "github.com/Wihrt/gatus-controller/api/v1alpha1"
)

// GatusEndpointValidator validates GatusEndpoint resources.
type GatusEndpointValidator struct{}

var (
	// validPlaceholders lists all Gatus-supported condition placeholders.
	validPlaceholders = []string{
		"STATUS",
		"RESPONSE_TIME",
		"IP",
		"BODY",
		"CONNECTED",
		"CERTIFICATE_EXPIRATION",
		"DOMAIN_EXPIRATION",
		"DNS_RCODE",
		"CONTEXT",
	}

	// placeholderRegex matches any [PLACEHOLDER] token in a condition string.
	// Gatus resolves placeholders case-insensitively (config/endpoint/placeholder.go:135),
	// so lowercase names are matched too and normalised before comparison.
	placeholderRegex = regexp.MustCompile(`\[[A-Za-z_]+\]`)

	// operatorRegex checks that at least one comparison operator is present.
	// Gatus only recognises operators surrounded by single spaces
	// (config/endpoint/condition.go:48-81), so "[STATUS]==200" is invalid.
	operatorRegex = regexp.MustCompile(` == | != | <= | >= | > | < `)

	// lenHasFuncRegex matches any len(...) or has(...) function call in an expression.
	lenHasFuncRegex = regexp.MustCompile(`(?i)\b(?:len|has)\(([^)]*)\)`)
)

var _ admission.Validator[*monitoringv1alpha1.GatusEndpoint] = &GatusEndpointValidator{}

func (v *GatusEndpointValidator) ValidateCreate(_ context.Context, ep *monitoringv1alpha1.GatusEndpoint) (admission.Warnings, error) {
	return nil, validateEndpointConditions(ep.Spec.Conditions)
}

func (v *GatusEndpointValidator) ValidateUpdate(_ context.Context, _, ep *monitoringv1alpha1.GatusEndpoint) (admission.Warnings, error) {
	return nil, validateEndpointConditions(ep.Spec.Conditions)
}

func (v *GatusEndpointValidator) ValidateDelete(_ context.Context, _ *monitoringv1alpha1.GatusEndpoint) (admission.Warnings, error) {
	return nil, nil
}

// validateEndpointConditions validates all condition strings in a GatusEndpoint spec.
func validateEndpointConditions(conditions []string) error {
	var errs field.ErrorList
	for i, c := range conditions {
		fld := field.NewPath("spec").Child("conditions").Index(i)
		if err := validateCondition(c, fld); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs.ToAggregate()
	}
	return nil
}

// validateCondition checks that a single condition string is syntactically valid.
func validateCondition(s string, fld *field.Path) *field.Error {
	if strings.TrimSpace(s) == "" {
		return field.Invalid(fld, s, "condition must not be empty")
	}

	// Check that a valid placeholder is present.
	matches := placeholderRegex.FindAllString(s, -1)
	if len(matches) == 0 {
		return field.Invalid(fld, s, "condition must contain a placeholder such as [STATUS] or [BODY]")
	}

	for _, m := range matches {
		// Strip brackets.
		name := strings.ToUpper(m[1 : len(m)-1])
		isValid := false
		for _, p := range validPlaceholders {
			if name == p {
				isValid = true
				break
			}
		}
		if !isValid {
			return field.Invalid(fld, s, fmt.Sprintf("condition contains an unknown placeholder %q; valid placeholders are %v", m, validPlaceholders))
		}
	}

	// Validate function usage: len() and has() require a valid placeholder as argument.
	for _, match := range lenHasFuncRegex.FindAllStringSubmatch(s, -1) {
		arg := strings.TrimSpace(match[1])
		hasValidPlaceholder := false
		for _, p := range validPlaceholders {
			if strings.HasPrefix(strings.ToUpper(arg), "["+p+"]") {
				hasValidPlaceholder = true
				break
			}
		}
		if !hasValidPlaceholder {
			return field.Invalid(fld, s, "len() and has() functions require a valid placeholder as argument (e.g. [BODY], [STATUS])")
		}
	}

	// Check that a comparison operator is present.
	if !operatorRegex.MatchString(s) {
		return field.Invalid(fld, s, "condition must contain a comparison operator surrounded by spaces (' == ', ' != ', ' < ', ' <= ', ' > ', ' >= ')")
	}

	return nil
}
