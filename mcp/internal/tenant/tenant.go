// Package tenant resolves and bootstraps the operating tenant context for the
// MCP binary. It reads MCP_ORG_SLUG and MCP_USER_SLUG from the environment and
// idempotently ensures the org, default campaign, user, and memberships exist in
// the database before returning a Tenant with the resolved IDs.
//
// The SSE server does not call Bootstrap — tenant context there comes from the
// authenticated request (future SaaS work).
package tenant

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

const (
	defaultOrgSlug  = "system_default_org"
	defaultUserSlug = "system_default_user"
)

// Tenant holds the resolved IDs for the operating tenant context.
type Tenant struct {
	OrgID      int32
	CampaignID int32
	UserID     int32
}

// Bootstrap idempotently ensures the org, default campaign, user, and memberships
// described by MCP_ORG_SLUG / MCP_USER_SLUG exist, then returns the resolved
// Tenant. Each step is checked independently so partial states (e.g. org exists
// but user does not) are handled correctly.
//
// Env vars:
//
//	MCP_ORG_SLUG  — slug of the operating org  (default: system_default_org)
//	MCP_USER_SLUG — slug of the operating user (default: system_default_user)
func Bootstrap(ctx context.Context, q *db.Queries) (*Tenant, error) {
	orgSlug := os.Getenv("MCP_ORG_SLUG")
	if orgSlug == "" {
		orgSlug = defaultOrgSlug
	}
	userSlug := os.Getenv("MCP_USER_SLUG")
	if userSlug == "" {
		userSlug = defaultUserSlug
	}

	// ── 1. Org ───────────────────────────────────────────────────────────────
	orgRow, err := q.GetOrgBySlug(ctx, orgSlug)
	var orgID int32
	if errors.Is(err, pgx.ErrNoRows) {
		created, cerr := q.CreateOrganization(ctx, db.CreateOrganizationParams{
			Name:     orgSlug,
			Slug:     orgSlug,
			IsSystem: orgSlug == defaultOrgSlug,
		})
		if cerr != nil {
			return nil, fmt.Errorf("create org %q: %w", orgSlug, cerr)
		}
		orgID = created.ID
	} else if err != nil {
		return nil, fmt.Errorf("get org %q: %w", orgSlug, err)
	} else {
		orgID = orgRow.ID
	}

	// ── 2. Default campaign ──────────────────────────────────────────────────
	// CreateOrganization does not automatically create the campaign — that
	// convention is enforced here and in the create_organization MCP tool so
	// both paths produce the same result.
	campaignRow, err := q.GetDefaultCampaignForOrg(ctx, orgID)
	var campaignID int32
	if errors.Is(err, pgx.ErrNoRows) {
		created, cerr := q.CreateCampaign(ctx, db.CreateCampaignParams{
			OrganizationID: orgID,
			Name:           "Default",
			Slug:           "default",
			IsDefault:      true,
		})
		if cerr != nil {
			return nil, fmt.Errorf("create default campaign for org %d: %w", orgID, cerr)
		}
		campaignID = created.ID
	} else if err != nil {
		return nil, fmt.Errorf("get default campaign for org %d: %w", orgID, err)
	} else {
		campaignID = campaignRow.ID
	}

	// ── 3. User ──────────────────────────────────────────────────────────────
	userRow, err := q.GetUserBySlug(ctx, userSlug)
	var userID int32
	if errors.Is(err, pgx.ErrNoRows) {
		created, cerr := q.CreateUser(ctx, db.CreateUserParams{
			Name:     userSlug,
			Slug:     userSlug,
			Email:    nil,
			IsSystem: userSlug == defaultUserSlug,
		})
		if cerr != nil {
			return nil, fmt.Errorf("create user %q: %w", userSlug, cerr)
		}
		userID = created.ID
	} else if err != nil {
		return nil, fmt.Errorf("get user %q: %w", userSlug, err)
	} else {
		userID = userRow.ID
	}

	// ── 4. Org membership ────────────────────────────────────────────────────
	if err := q.AddOrgMember(ctx, db.AddOrgMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           db.AppOrgRoleOwner,
	}); err != nil {
		return nil, fmt.Errorf("add org member: %w", err)
	}

	// ── 5. Campaign membership ───────────────────────────────────────────────
	if err := q.AddCampaignMember(ctx, db.AddCampaignMemberParams{
		CampaignID: campaignID,
		UserID:     userID,
		Role:       db.AppCampaignRoleAdmin,
	}); err != nil {
		return nil, fmt.Errorf("add campaign member: %w", err)
	}

	return &Tenant{
		OrgID:      orgID,
		CampaignID: campaignID,
		UserID:     userID,
	}, nil
}
