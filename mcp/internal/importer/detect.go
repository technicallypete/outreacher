package importer

import (
	"fmt"
	"strings"
)

// DetectFormat inspects the CSV header row and returns the format identifier.
// Returns an error if the header doesn't match any known format.
func DetectFormat(content string) (string, error) {
	// Extract just the first line for header inspection.
	firstLine := content
	if i := strings.Index(content, "\n"); i >= 0 {
		firstLine = content[:i]
	}
	firstLine = strings.ToLower(firstLine)

	has := func(col string) bool {
		return strings.Contains(firstLine, strings.ToLower(col))
	}

	// Gojiberry: unique columns are "Intent" and "Total Score".
	if has("intent keyword") || has("total score") || has("personnalized") {
		return "gojiberry", nil
	}

	// Revli investor formats: unique columns are "Firm Type", "Stage Focus",
	// "Investor Thesis", "Portfolio Size".
	isInvestor := has("firm type") || has("investor thesis") || has("stage focus") || has("portfolio size")

	// Revli startup formats: unique columns are "Recent Funding Amount",
	// "Key Technical Challenges", "Challenges And Risks".
	isStartup := has("recent funding amount") || has("key technical challenges") || has("challenges and risks")

	// Contact rows have "First Name" / "Last Name"; company-only rows do not
	// (startup_company has "Company Name" but no "First Name").
	hasContacts := has("first name") || has("last name")

	switch {
	case isInvestor && hasContacts:
		return "revli_investor_contacts", nil
	case isInvestor && !hasContacts:
		return "revli_investor_companies", nil
	case isStartup && hasContacts:
		return "revli_startup_contacts", nil
	case isStartup && !hasContacts:
		return "revli_startup_companies", nil
	}

	return "", fmt.Errorf("could not detect CSV format from headers — pass format explicitly (gojiberry, revli_startup_contacts, revli_investor_contacts, revli_startup_companies, revli_investor_companies)")
}
