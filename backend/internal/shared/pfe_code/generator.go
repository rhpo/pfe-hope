package pfe_code

import (
	"fmt"
	"strings"
)

func Generate(specialityCode string, academicYearLabel string, sequence int) string {

	year := extractEndYear(academicYearLabel)
	return fmt.Sprintf("PFE-%s-%s-%03d", strings.ToUpper(specialityCode), year, sequence)
}

func extractEndYear(label string) string {
	parts := strings.Split(label, "-")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1])
	}
	return label
}
