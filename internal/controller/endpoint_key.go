package controller

import "strings"

// gatusKeyReplacer mirrors the character replacements done by Gatus' sanitize
// (config/key/key.go in TwiN/gatus).
var gatusKeyReplacer = strings.NewReplacer(
	"/", "-",
	"_", "-",
	".", "-",
	",", "-",
	" ", "-",
	"#", "-",
	"+", "-",
	"&", "-",
)

func sanitizeGatusKey(s string) string {
	return gatusKeyReplacer.Replace(strings.TrimSpace(strings.ToLower(s)))
}

// gatusEndpointKey computes the unique key Gatus derives from an endpoint's
// group and name (key.ConvertGroupAndNameToKey upstream). Gatus rejects the
// whole configuration when two endpoints or external endpoints share a key.
func gatusEndpointKey(group, name string) string {
	return sanitizeGatusKey(group) + "_" + sanitizeGatusKey(name)
}
