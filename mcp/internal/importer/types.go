package importer

import "time"

// NormalizedCompany is the common intermediate representation for a company
// from any import source. Nil pointer fields are omitted on upsert (COALESCE
// in the SQL preserves existing values).
type NormalizedCompany struct {
	Name               string
	Domain             *string
	Industry           *string
	LinkedinURL        *string
	Description        *string
	Headquarters       *string
	Phone              *string
	TwitterURL         *string
	FacebookURL        *string
	EmployeeCount      *int32
	FoundedDate        *string // YYYY-MM-DD
	AnnualRevenue      *string // "$10M to $50M"
	AnnualRevenueDate  *string // YYYY-MM-DD
	Technologies       *string
	FundingStage       *string
	FundingStatus      *string
	FundingAmountLast  *int64
	FundingDateLast    *string // YYYY-MM-DD
	FundingAmountTotal *int64
	TopInvestors       *string
	IsHiring           *bool
	IsVC               bool
	FirmType           *string
	StageFocus         *string
	CheckSize          *string
	PortfolioSize      *int32
	IndustryFocus      *string
	GeographyFocus     *string
	Intel              *string // JSON blob of AI-generated narrative fields
}

// NormalizedLead is the common intermediate representation for a contact/lead.
type NormalizedLead struct {
	FirstName   string
	LastName    string
	Email       *string
	EmailStatus *string
	LinkedinURL *string
	Title       *string
	Department  *string
	Phone       *string
	Location    *string // city, state, country joined
	Score       *float32
	SourcedAt   *time.Time // date from the source system (e.g. Gojiberry "Import Date")
}

// Row pairs a normalized lead with their company. Lead is nil for company-only
// imports (e.g. Revli startup_company and investor_company exports).
type Row struct {
	Company *NormalizedCompany
	Lead    *NormalizedLead
}

// ImportedRow is one lead entry in an ImportSummary.
type ImportedRow struct {
	ID      int32  `json:"id"`
	Name    string `json:"name"`
	Company string `json:"company,omitempty"`
	Email   string `json:"email,omitempty"`
	Action  string `json:"action"` // "created" or "updated"
}

// ImportSummary is returned by Write after all rows are processed.
type ImportSummary struct {
	Companies int           `json:"companies"`
	Leads     int           `json:"leads"`
	Skipped   int           `json:"skipped"`
	Rows      []ImportedRow `json:"rows,omitempty"`
}
