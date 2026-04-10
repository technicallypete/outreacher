package importer

import "fmt"

// ParseRevliInvestorContacts parses a Revli investor_contacts CSV export.
// Each row yields a NormalizedCompany (VC firm, is_vc=true) and a NormalizedLead.
func ParseRevliInvestorContacts(content string) ([]Row, error) {
	col, rows, err := parseCSV(content)
	if err != nil {
		return nil, err
	}

	out := make([]Row, 0, len(rows))
	for i, row := range rows {
		company := revliInvestorCompanyFromRow(row, col)
		lead := revliInvestorLeadFromRow(row, col)
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

// ParseRevliInvestorCompanies parses a Revli investor_company CSV export.
// Each row yields only a NormalizedCompany (no lead/contact data).
func ParseRevliInvestorCompanies(content string) ([]Row, error) {
	col, rows, err := parseCSV(content)
	if err != nil {
		return nil, err
	}

	out := make([]Row, 0, len(rows))
	for i, row := range rows {
		company := revliInvestorCompanyFromRow(row, col)
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

// revliInvestorCompanyFromRow extracts the VC firm fields from an investor row.
func revliInvestorCompanyFromRow(row []string, col map[string]int) *NormalizedCompany {
	name := get(row, col, "Company Name")
	if name == "" {
		return nil
	}

	// Portfolio size may be stored as a plain integer.
	portfolioSize := nullableInt32(get(row, col, "Portfolio Size"))

	intel := buildIntel(
		"firm_overview", get(row, col, "Firm Overview"),
		"investor_thesis", get(row, col, "Investor Thesis"),
		"portfolio_highlights", get(row, col, "Portfolio Highlights"),
		"co_investor_network", get(row, col, "Co-Investor Network"),
		"startups_invested_this_week", get(row, col, "Startups Invested This Week"),
		"historical_investment", get(row, col, "Historical Investment"),
	)

	return &NormalizedCompany{
		Name:           name,
		Domain:         nullable(get(row, col, "Website")),
		LinkedinURL:    nullable(get(row, col, "Company Linkedin Url")),
		Description:    nullable(get(row, col, "Company Description")),
		Headquarters: joinLocation(
			get(row, col, "Company City"),
			get(row, col, "Company State"),
			get(row, col, "Company Country"),
		),
		Phone:         nullable(get(row, col, "Company Phone")),
		TwitterURL:    nullable(get(row, col, "Twitter Url")),
		FacebookURL:   nullable(get(row, col, "Facebook Url")),
		EmployeeCount: nullableInt32(get(row, col, "# Employees")),
		FoundedDate:   nullable(get(row, col, "Founded Date")),
		IsVC:          true,
		FirmType:      nullable(get(row, col, "Firm Type")),
		StageFocus:    nullable(get(row, col, "Stage Focus")),
		CheckSize:     nullable(get(row, col, "Check Size")),
		PortfolioSize: portfolioSize,
		IndustryFocus: nullable(get(row, col, "Industry Focus")),
		GeographyFocus: nullable(get(row, col, "Geography Focus")),
		Intel:         intel,
	}
}

// revliInvestorLeadFromRow extracts contact fields from an investor_contacts row.
func revliInvestorLeadFromRow(row []string, col map[string]int) *NormalizedLead {
	firstName := get(row, col, "First Name")
	lastName := get(row, col, "Last Name")
	email := get(row, col, "Contact Email")
	linkedin := get(row, col, "Person Linkedin Url")

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
		Department:  nullable(get(row, col, "Job Department")),
		Phone:       nullable(get(row, col, "Contact Mobile")),
		Location: joinLocation(
			get(row, col, "Contact City"),
			get(row, col, "Contact State"),
			get(row, col, "Contact Country"),
		),
	}
}
