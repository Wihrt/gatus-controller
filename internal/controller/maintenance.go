package controller

import "strings"

// gatusDayNames maps lowercase day names to the capitalised form Gatus requires.
var gatusDayNames = map[string]string{
	"sunday": "Sunday", "monday": "Monday", "tuesday": "Tuesday", "wednesday": "Wednesday",
	"thursday": "Thursday", "friday": "Friday", "saturday": "Saturday",
}

// maintenanceDays builds the Gatus "every" list from the deprecated day field
// followed by every. Names are trimmed and capitalised (input is case-insensitive),
// duplicates are dropped and order is preserved. Unknown names are passed through
// unchanged (the CRD schema rejects them).
func maintenanceDays(day string, every []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range append([]string{day}, every...) {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if n, ok := gatusDayNames[strings.ToLower(d)]; ok {
			d = n
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}
