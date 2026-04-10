-- name: UpsertCompany :one
-- Upsert a company within a campaign. Conflicts on (campaign_id, name).
-- COALESCE preserves existing values when re-importing with partial data.
-- is_vc is sticky: once true it stays true even if a later import omits it.
INSERT INTO app.companies (
    campaign_id, name, domain, industry, linkedin_url,
    description, headquarters, phone, twitter_url, facebook_url,
    employee_count, founded_date,
    annual_revenue, annual_revenue_date,
    technologies, funding_stage, funding_status,
    funding_amount_last, funding_date_last, funding_amount_total,
    top_investors, is_hiring, is_vc,
    firm_type, stage_focus, check_size, portfolio_size,
    industry_focus, geography_focus, intel
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12,
    $13, $14,
    $15, $16, $17,
    $18, $19, $20,
    $21, $22, $23,
    $24, $25, $26, $27,
    $28, $29, $30
)
ON CONFLICT (campaign_id, name) DO UPDATE SET
    domain              = COALESCE(EXCLUDED.domain,              app.companies.domain),
    industry            = COALESCE(EXCLUDED.industry,            app.companies.industry),
    linkedin_url        = COALESCE(EXCLUDED.linkedin_url,        app.companies.linkedin_url),
    description         = COALESCE(EXCLUDED.description,         app.companies.description),
    headquarters        = COALESCE(EXCLUDED.headquarters,        app.companies.headquarters),
    phone               = COALESCE(EXCLUDED.phone,               app.companies.phone),
    twitter_url         = COALESCE(EXCLUDED.twitter_url,         app.companies.twitter_url),
    facebook_url        = COALESCE(EXCLUDED.facebook_url,        app.companies.facebook_url),
    employee_count      = COALESCE(EXCLUDED.employee_count,      app.companies.employee_count),
    founded_date        = COALESCE(EXCLUDED.founded_date,        app.companies.founded_date),
    annual_revenue      = COALESCE(EXCLUDED.annual_revenue,      app.companies.annual_revenue),
    annual_revenue_date = COALESCE(EXCLUDED.annual_revenue_date, app.companies.annual_revenue_date),
    technologies        = COALESCE(EXCLUDED.technologies,        app.companies.technologies),
    funding_stage       = COALESCE(EXCLUDED.funding_stage,       app.companies.funding_stage),
    funding_status      = COALESCE(EXCLUDED.funding_status,      app.companies.funding_status),
    funding_amount_last  = COALESCE(EXCLUDED.funding_amount_last,  app.companies.funding_amount_last),
    funding_date_last    = COALESCE(EXCLUDED.funding_date_last,    app.companies.funding_date_last),
    funding_amount_total = COALESCE(EXCLUDED.funding_amount_total, app.companies.funding_amount_total),
    top_investors       = COALESCE(EXCLUDED.top_investors,       app.companies.top_investors),
    is_hiring           = COALESCE(EXCLUDED.is_hiring,           app.companies.is_hiring),
    is_vc               = EXCLUDED.is_vc OR app.companies.is_vc,
    firm_type           = COALESCE(EXCLUDED.firm_type,           app.companies.firm_type),
    stage_focus         = COALESCE(EXCLUDED.stage_focus,         app.companies.stage_focus),
    check_size          = COALESCE(EXCLUDED.check_size,          app.companies.check_size),
    portfolio_size      = COALESCE(EXCLUDED.portfolio_size,      app.companies.portfolio_size),
    industry_focus      = COALESCE(EXCLUDED.industry_focus,      app.companies.industry_focus),
    geography_focus     = COALESCE(EXCLUDED.geography_focus,     app.companies.geography_focus),
    intel               = COALESCE(EXCLUDED.intel,               app.companies.intel)
RETURNING id;

-- name: GetCompany :one
-- Fetch full company detail by id, scoped to campaign. Returns all enriched fields.
SELECT
    id, name, domain, industry, linkedin_url,
    description, headquarters, phone, twitter_url, facebook_url,
    employee_count, founded_date,
    annual_revenue, annual_revenue_date,
    technologies, funding_stage, funding_status,
    funding_amount_last, funding_date_last, funding_amount_total,
    top_investors, is_hiring, is_vc,
    firm_type, stage_focus, check_size, portfolio_size,
    industry_focus, geography_focus, intel, campaign_id,
    created_at, updated_at
FROM app.companies
WHERE id = $1 AND campaign_id = $2;

-- name: SearchCompanies :many
-- Search companies within a campaign. All filters are optional and AND-ed together.
-- Pass empty string to skip a text filter; 'true'/'false' for boolean filters.
SELECT
    id, name, domain, industry, linkedin_url, headquarters,
    description, employee_count, funding_stage, funding_status,
    funding_amount_last, funding_date_last, annual_revenue,
    is_hiring, is_vc, firm_type, stage_focus, check_size, campaign_id
FROM app.companies
WHERE
    campaign_id = $1
    AND ($2 = '' OR name ILIKE '%' || $2 || '%')
    AND ($3 = '' OR is_vc::text = $3)
    AND ($4 = '' OR COALESCE(funding_stage, '') ILIKE '%' || $4 || '%')
    AND ($5 = '' OR COALESCE(industry, '') ILIKE '%' || $5 || '%')
    AND ($6 = '' OR (is_hiring IS NOT NULL AND is_hiring::text = $6))
ORDER BY name
LIMIT 50;
