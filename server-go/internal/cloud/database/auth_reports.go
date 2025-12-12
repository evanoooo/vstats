package database

import (
	"context"
	"time"

	"vstats/internal/cloud/models"
)

// ============================================================================
// Auth Reports Operations
// ============================================================================

// CreateAuthReport creates a new auth report
func CreateAuthReport(ctx context.Context, report *models.AuthReport) error {
	report.ReportedAt = time.Now()

	_, err := pool.Exec(ctx, `
		INSERT INTO auth_reports (site_url, site_host, provider, username, ip_address, user_agent, reported_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, report.SiteURL, report.SiteHost, report.Provider, report.Username, report.IPAddress, report.UserAgent, report.ReportedAt)

	return err
}

// GetAuthDailyStats returns daily auth statistics
func GetAuthDailyStats(ctx context.Context, days int) ([]models.AuthDailyStats, error) {
	if days <= 0 {
		days = 30
	}

	rows, err := pool.Query(ctx, `
		SELECT 
			reported_at::DATE::TEXT AS date,
			COUNT(DISTINCT site_host) AS unique_sites,
			COUNT(DISTINCT username) AS unique_users,
			COUNT(*) AS total_auths,
			COUNT(DISTINCT CASE WHEN provider = 'github' THEN username END) AS github_users,
			COUNT(DISTINCT CASE WHEN provider = 'google' THEN username END) AS google_users
		FROM auth_reports
		WHERE reported_at >= CURRENT_DATE - $1 * INTERVAL '1 day'
		GROUP BY reported_at::DATE
		ORDER BY date DESC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.AuthDailyStats
	for rows.Next() {
		var s models.AuthDailyStats
		if err := rows.Scan(&s.Date, &s.UniqueSites, &s.UniqueUsers, &s.TotalAuths, &s.GitHubUsers, &s.GoogleUsers); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetAuthSiteStats returns site statistics
func GetAuthSiteStats(ctx context.Context, limit int) ([]models.AuthSiteStats, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := pool.Query(ctx, `
		SELECT 
			site_host,
			MAX(site_url) AS site_url,
			COUNT(DISTINCT username) AS unique_users,
			COUNT(*) AS total_auths,
			MIN(reported_at) AS first_seen,
			MAX(reported_at) AS last_seen,
			COUNT(DISTINCT reported_at::DATE) AS active_days
		FROM auth_reports
		GROUP BY site_host
		ORDER BY last_seen DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.AuthSiteStats
	for rows.Next() {
		var s models.AuthSiteStats
		if err := rows.Scan(&s.SiteHost, &s.SiteURL, &s.UniqueUsers, &s.TotalAuths, &s.FirstSeen, &s.LastSeen, &s.ActiveDays); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetAuthOverallStats returns overall auth statistics
func GetAuthOverallStats(ctx context.Context) (*models.AuthOverallStats, error) {
	var stats models.AuthOverallStats

	// Overall stats
	err := pool.QueryRow(ctx, `
		SELECT 
			COUNT(DISTINCT site_host) AS total_sites,
			COUNT(DISTINCT username) AS total_users,
			COUNT(*) AS total_auths,
			COUNT(DISTINCT CASE WHEN provider = 'github' THEN username END) AS github_users,
			COUNT(DISTINCT CASE WHEN provider = 'google' THEN username END) AS google_users
		FROM auth_reports
	`).Scan(&stats.TotalSites, &stats.TotalUsers, &stats.TotalAuths, &stats.GitHubUsers, &stats.GoogleUsers)
	if err != nil {
		return nil, err
	}

	// Today stats
	err = pool.QueryRow(ctx, `
		SELECT 
			COUNT(DISTINCT site_host) AS today_sites,
			COUNT(DISTINCT username) AS today_users,
			COUNT(*) AS today_auths
		FROM auth_reports
		WHERE reported_at::DATE = CURRENT_DATE
	`).Scan(&stats.TodaySites, &stats.TodayUsers, &stats.TodayAuths)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAuthUsersBySite returns users for a specific site
func GetAuthUsersBySite(ctx context.Context, siteHost string, limit int) ([]models.AuthReport, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := pool.Query(ctx, `
		SELECT id, site_url, site_host, provider, username, ip_address, user_agent, reported_at
		FROM auth_reports
		WHERE site_host = $1
		ORDER BY reported_at DESC
		LIMIT $2
	`, siteHost, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.AuthReport
	for rows.Next() {
		var r models.AuthReport
		if err := rows.Scan(&r.ID, &r.SiteURL, &r.SiteHost, &r.Provider, &r.Username, &r.IPAddress, &r.UserAgent, &r.ReportedAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}

	return reports, nil
}

// GetAuthUsersByDate returns auth reports for a specific date
func GetAuthUsersByDate(ctx context.Context, date string, limit int) ([]models.AuthReport, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := pool.Query(ctx, `
		SELECT id, site_url, site_host, provider, username, ip_address, user_agent, reported_at
		FROM auth_reports
		WHERE reported_at::DATE = $1::DATE
		ORDER BY reported_at DESC
		LIMIT $2
	`, date, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.AuthReport
	for rows.Next() {
		var r models.AuthReport
		if err := rows.Scan(&r.ID, &r.SiteURL, &r.SiteHost, &r.Provider, &r.Username, &r.IPAddress, &r.UserAgent, &r.ReportedAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}

	return reports, nil
}

