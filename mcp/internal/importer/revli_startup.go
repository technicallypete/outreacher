package importer

import "fmt"

// ParseRevliStartupContacts parses a Revli startup_contacts CSV export.
// Each row yields a NormalizedCompany (with full enrichment) and a NormalizedLead.
func ParseRevliStartupContacts(content string) ([]Row, error) {
	col, rows, err := parseCSV(content)
	if err != nil {
		return nil, err
	}

	out := make([]Row, 0, len(rows))
	for i, row := range rows {
		company := revliStartupCompanyFromRow(row, col)
		lead := revliStartupLeadFromRow(row, col)
		if company == nil && lead == nil {
			continue
		}
		if company != nil && company.Name == "" {
			return nil, fmt.Errorf("row %d: Company Name is required", i+2)
		}
		out = append(out, Row{Company: company, Lead: lead})
	}
	return out, nil
}

// ParseRevliStartupCompanies parses a Revli startup_company CSV export.
// Each row yields only a NormalizedCompany (no lead/contact data).
func ParseRevliStartupCompanies(content string) ([]Row, error) {
	col, rows, err := parseCSV(content)
	if err != nil {
		return nil, err
	}

	out := make([]Row, 0, len(rows))
	for i, row := range rows {
		company := revliStartupCompanyFromRow(row, col)
		if company == nil {
			continue
		}
		if company.Name == "" {
			return nil, fmt.Errorf("row %d: Company Name is required", i+2)
		}
		out = append(out, Row{Company: company})
	}
	return out, nil
}

// revliStartupCompanyFromRow extracts the company fields from a startup row.
func revliStartupCompanyFromRow(row []string, col map[string]int) *NormalizedCompany {
	name := get(row, col, "Company Name")
	if name == "" {
		return nil
	}

	// Derive is_hiring: non-empty "Currently Hiring" field means active jobs.
	var isHiring *bool
	if h := get(row, col, "Currently Hiring"); h != "" {
		isHiring = boolPtr(true)
	} else {
		isHiring = boolPtr(false)
	}

	// Extract domain from full website URL.
	website := get(row, col, "Company Website")

	intel := buildIntel(
		"overview", get(row, col, "Company Overview"),
		"description", get(row, col, "Company Description"),
		"expansion_strategy", get(row, col, "Expansion Strategy"),
		"market_opportunities", get(row, col, "Market Opportunities"),
		"challenges_risks", get(row, col, "Challenges And Risks"),
		"tech_needs", get(row, col, "Tech Needs"),
		"infrastructure_needs", get(row, col, "Infrastructure Needs"),
		"service_needs", get(row, col, "Service Needs"),
		"projected_growth", get(row, col, "Projected Growth"),
		"hiring_forecast", get(row, col, "Hiring Forecast"),
		"key_technical_challenges", get(row, col, "Key Technical Challenges"),
		"funding_news", get(row, col, "Funding News"),
	)

	return &NormalizedCompany{
		Name:               name,
		Domain:             nullable(website),
		Industry:           nullable(get(row, col, "Industries")),
		LinkedinURL:        nullable(get(row, col, "Company LinkedIn")),
		Description:        nullable(get(row, col, "Company Description")),
		Headquarters:       nullable(get(row, col, "Company Headquarters")),
		Phone:              nullable(get(row, col, "Company Phone")),
		TwitterURL:         nullable(get(row, col, "Company Twitter")),
		FacebookURL:        nullable(get(row, col, "Company Facebook")),
		EmployeeCount:      nullableInt32(get(row, col, "# Employees")),
		FoundedDate:        nullable(get(row, col, "Founded Date")),
		AnnualRevenue:      nullable(get(row, col, "Annual Revenue")),
		Technologies:       nullable(get(row, col, "Company Technologies")),
		FundingStage:       nullable(get(row, col, "Last Funding Type")),
		FundingStatus:      nullable(get(row, col, "Funding Status")),
		FundingAmountLast:  parseDollarAmount(get(row, col, "Recent Funding Amount(USD)")),
		FundingDateLast:    nullable(get(row, col, "Recent Funding Date")),
		FundingAmountTotal: parseDollarAmount(get(row, col, "Total Funding Amount(USD)")),
		TopInvestors:       nullable(get(row, col, "Top 5 Investors")),
		IsHiring:           isHiring,
		IsVC:               false,
		Intel:              intel,
	}
}

// revliStartupLeadFromRow extracts contact fields from a startup_contacts row.
func revliStartupLeadFromRow(row []string, col map[string]int) *NormalizedLead {
	firstName := get(row, col, "First Name")
	lastName := get(row, col, "Last Name")
	email := get(row, col, "Email")
	linkedin := get(row, col, "Contact LinkedIn")

	if firstName == "" && lastName == "" && email == "" && linkedin == "" {
		return nil
	}

	return &NormalizedLead{
		FirstName:   firstName,
		LastName:    lastName,
		Email:       nullable(email),
		EmailStatus: nullable(get(row, col, "Email Status")),
		LinkedinURL: nullable(linkedin),
		Title:       nullable(get(row, col, "Job Title")),
		Department:  nullable(get(row, col, "Job Departments")),
		Phone:       nullable(get(row, col, "Contact Mobile")),
		Location: joinLocation(
			get(row, col, "Contact City"),
			get(row, col, "Contact State"),
			get(row, col, "Contact Country"),
		),
	}
}
