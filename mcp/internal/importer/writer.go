package importer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// Write upserts all rows into the database under campaignID. Company-only rows
// (Lead == nil) are written as companies with no lead. Returns a summary of
// what was created/updated.
func Write(ctx context.Context, q *db.Queries, rows []Row, campaignID int32) (ImportSummary, error) {
	var summary ImportSummary
	seenCompanies := make(map[string]int32)

	for _, row := range rows {
		var companyID *int32

		// ── Company ──────────────────────────────────────────────────────────
		if row.Company != nil {
			c := row.Company
			cid, exists := seenCompanies[c.Name]
			if !exists {
				var err error
				cid, err = q.UpsertCompany(ctx, db.UpsertCompanyParams{
					CampaignID:         campaignID,
					Name:               c.Name,
					Domain:             c.Domain,
					Industry:           c.Industry,
					LinkedinUrl:        c.LinkedinURL,
					Description:        c.Description,
					Headquarters:       c.Headquarters,
					Phone:              c.Phone,
					TwitterUrl:         c.TwitterURL,
					FacebookUrl:        c.FacebookURL,
					EmployeeCount:      c.EmployeeCount,
					FoundedDate:        c.FoundedDate,
					AnnualRevenue:      c.AnnualRevenue,
					AnnualRevenueDate:  c.AnnualRevenueDate,
					Technologies:       c.Technologies,
					FundingStage:       c.FundingStage,
					FundingStatus:      c.FundingStatus,
					FundingAmountLast:  c.FundingAmountLast,
					FundingDateLast:    c.FundingDateLast,
					FundingAmountTotal: c.FundingAmountTotal,
					TopInvestors:       c.TopInvestors,
					IsHiring:           c.IsHiring,
					IsVc:               c.IsVC,
					FirmType:           c.FirmType,
					StageFocus:         c.StageFocus,
					CheckSize:          c.CheckSize,
					PortfolioSize:      c.PortfolioSize,
					IndustryFocus:      c.IndustryFocus,
					GeographyFocus:     c.GeographyFocus,
					Intel:              c.Intel,
				})
				if err != nil {
					return summary, fmt.Errorf("upsert company %q: %w", c.Name, err)
				}
				seenCompanies[c.Name] = cid
				summary.Companies++
			}
			companyID = &cid
		}

		// ── Lead ─────────────────────────────────────────────────────────────
		if row.Lead == nil {
			continue
		}
		l := row.Lead

		name := strings.TrimSpace(l.FirstName + " " + l.LastName)
		if name == "" {
			if l.Email != nil {
				name = *l.Email
			} else if l.LinkedinURL != nil {
				name = *l.LinkedinURL
			} else {
				summary.Skipped++
				continue
			}
		}

		// Determine primary identifier for dedup.
		var identType, identValue string
		switch {
		case l.Email != nil && *l.Email != "":
			identType, identValue = "email", *l.Email
		case l.LinkedinURL != nil && *l.LinkedinURL != "":
			identType, identValue = "linkedin", *l.LinkedinURL
		default:
			summary.Skipped++
			continue
		}

		leadID, err := q.FindLeadByIdentifier(ctx, db.FindLeadByIdentifierParams{
			CampaignID: campaignID,
			Type:       identType,
			Value:      identValue,
		})

		sourcedAt := pgtype.Timestamptz{}
		if l.SourcedAt != nil {
			sourcedAt = pgtype.Timestamptz{Time: *l.SourcedAt, Valid: true}
		}

		action := "updated"
		if errors.Is(err, pgx.ErrNoRows) {
			leadID, err = q.CreateLead(ctx, db.CreateLeadParams{
				CampaignID:  campaignID,
				Name:        name,
				Email:       l.Email,
				CompanyID:   companyID,
				Title:       l.Title,
				Score:       l.Score,
				LinkedinUrl: l.LinkedinURL,
				Location:    l.Location,
				Phone:       l.Phone,
				EmailStatus: l.EmailStatus,
				Department:  l.Department,
				SourcedAt:   sourcedAt,
			})
			if err != nil {
				return summary, fmt.Errorf("create lead %q: %w", identValue, err)
			}
			summary.Leads++
			action = "created"
		} else if err != nil {
			return summary, fmt.Errorf("find lead by identifier: %w", err)
		} else {
			if err := q.UpdateLeadFields(ctx, db.UpdateLeadFieldsParams{
				ID:          leadID,
				CampaignID:  campaignID,
				Name:        name,
				Email:       l.Email,
				CompanyID:   companyID,
				Title:       l.Title,
				Score:       l.Score,
				LinkedinUrl: l.LinkedinURL,
				Location:    l.Location,
				Phone:       l.Phone,
				EmailStatus: l.EmailStatus,
				Department:  l.Department,
				SourcedAt:   sourcedAt,
			}); err != nil {
				return summary, fmt.Errorf("update lead %d: %w", leadID, err)
			}
		}

		irow := ImportedRow{ID: leadID, Name: name, Action: action}
		if l.Email != nil {
			irow.Email = *l.Email
		}
		if companyID != nil {
			for cname, cid := range seenCompanies {
				if cid == *companyID {
					irow.Company = cname
					break
				}
			}
		}
		summary.Rows = append(summary.Rows, irow)

		// Register primary identifier.
		if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
			CampaignID: campaignID,
			LeadID:     leadID,
			Type:       identType,
			Value:      identValue,
		}); err != nil {
			return summary, fmt.Errorf("upsert identifier: %w", err)
		}
		// Register secondary identifier if both are present.
		if identType == "email" && l.LinkedinURL != nil && *l.LinkedinURL != "" {
			if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
				CampaignID: campaignID,
				LeadID:     leadID,
				Type:       "linkedin",
				Value:      *l.LinkedinURL,
			}); err != nil {
				return summary, fmt.Errorf("upsert linkedin identifier: %w", err)
			}
		} else if identType == "linkedin" && l.Email != nil && *l.Email != "" {
			if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
				CampaignID: campaignID,
				LeadID:     leadID,
				Type:       "email",
				Value:      *l.Email,
			}); err != nil {
				return summary, fmt.Errorf("upsert email identifier: %w", err)
			}
		}
	}

	return summary, nil
}
