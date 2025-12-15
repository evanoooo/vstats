package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Audit Log Functions
// ============================================================================

// LogAudit creates an audit log entry
func LogAudit(entry AuditLogEntry) {
	if dbWriter == nil {
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	status := entry.Status
	if status == "" {
		status = "success"
	}

	dbWriter.WriteAsync(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO audit_logs (timestamp, action, category, user_ip, user_agent, target_type, target_id, target_name, details, status, error_message)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			timestamp,
			string(entry.Action),
			string(entry.Category),
			entry.UserIP,
			entry.UserAgent,
			entry.TargetType,
			entry.TargetID,
			entry.TargetName,
			entry.Details,
			status,
			entry.ErrorMessage,
		)
		return err
	})
}

// LogAuditFromContext creates an audit log entry using request context
func LogAuditFromContext(c *gin.Context, action AuditLogAction, category AuditLogCategory, targetType, targetID, targetName, details string) {
	LogAudit(AuditLogEntry{
		Action:     action,
		Category:   category,
		UserIP:     c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		TargetType: targetType,
		TargetID:   targetID,
		TargetName: targetName,
		Details:    details,
		Status:     "success",
	})
}

// LogAuditError creates an audit log entry for failed operations
func LogAuditError(c *gin.Context, action AuditLogAction, category AuditLogCategory, targetType, targetID, targetName, details, errorMessage string) {
	LogAudit(AuditLogEntry{
		Action:       action,
		Category:     category,
		UserIP:       c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		TargetType:   targetType,
		TargetID:     targetID,
		TargetName:   targetName,
		Details:      details,
		Status:       "error",
		ErrorMessage: errorMessage,
	})
}

// ============================================================================
// Audit Log Handlers
// ============================================================================

// GetAuditLogs returns paginated audit logs
func (s *AppState) GetAuditLogs(c *gin.Context) {
	var query AuditLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 200 {
		query.Limit = 200
	}

	db := dbWriter.GetDB()

	// Build query
	whereConditions := []string{}
	args := []interface{}{}

	if query.Category != "" {
		whereConditions = append(whereConditions, "category = ?")
		args = append(args, string(query.Category))
	}

	if query.Action != "" {
		whereConditions = append(whereConditions, "action = ?")
		args = append(args, string(query.Action))
	}

	if query.StartDate != "" {
		whereConditions = append(whereConditions, "timestamp >= ?")
		args = append(args, query.StartDate)
	}

	if query.EndDate != "" {
		whereConditions = append(whereConditions, "timestamp <= ?")
		args = append(args, query.EndDate)
	}

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		whereConditions = append(whereConditions, "(target_name LIKE ? OR details LIKE ? OR user_ip LIKE ?)")
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count audit logs"})
		return
	}

	// Get paginated results
	offset := (query.Page - 1) * query.Limit
	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, action, category, user_ip, COALESCE(user_agent, ''), 
		       COALESCE(target_type, ''), COALESCE(target_id, ''), COALESCE(target_name, ''),
		       COALESCE(details, ''), status, COALESCE(error_message, '')
		FROM audit_logs %s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`, whereClause)

	args = append(args, query.Limit, offset)
	rows, err := db.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(
			&log.ID, &log.Timestamp, &log.Action, &log.Category,
			&log.UserIP, &log.UserAgent, &log.TargetType, &log.TargetID,
			&log.TargetName, &log.Details, &log.Status, &log.ErrorMessage,
		); err != nil {
			continue
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	c.JSON(http.StatusOK, AuditLogResponse{
		Logs:  logs,
		Total: total,
		Page:  query.Page,
		Limit: query.Limit,
	})
}

// ExportAuditLogs exports audit logs as JSON or CSV
func (s *AppState) ExportAuditLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	
	var query AuditLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	db := dbWriter.GetDB()

	// Build query
	whereConditions := []string{}
	args := []interface{}{}

	if query.Category != "" {
		whereConditions = append(whereConditions, "category = ?")
		args = append(args, string(query.Category))
	}

	if query.Action != "" {
		whereConditions = append(whereConditions, "action = ?")
		args = append(args, string(query.Action))
	}

	if query.StartDate != "" {
		whereConditions = append(whereConditions, "timestamp >= ?")
		args = append(args, query.StartDate)
	}

	if query.EndDate != "" {
		whereConditions = append(whereConditions, "timestamp <= ?")
		args = append(args, query.EndDate)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get all matching results (limit to 10000 for safety)
	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, action, category, user_ip, COALESCE(user_agent, ''), 
		       COALESCE(target_type, ''), COALESCE(target_id, ''), COALESCE(target_name, ''),
		       COALESCE(details, ''), status, COALESCE(error_message, '')
		FROM audit_logs %s
		ORDER BY timestamp DESC
		LIMIT 10000`, whereClause)

	rows, err := db.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(
			&log.ID, &log.Timestamp, &log.Action, &log.Category,
			&log.UserIP, &log.UserAgent, &log.TargetType, &log.TargetID,
			&log.TargetName, &log.Details, &log.Status, &log.ErrorMessage,
		); err != nil {
			continue
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	filename := fmt.Sprintf("audit-logs-%s", time.Now().Format("2006-01-02"))

	if format == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
		
		// Write CSV header
		c.Writer.WriteString("ID,Timestamp,Action,Category,UserIP,UserAgent,TargetType,TargetID,TargetName,Details,Status,ErrorMessage\n")
		
		for _, log := range logs {
			line := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
				log.ID,
				escapeCSV(log.Timestamp),
				escapeCSV(string(log.Action)),
				escapeCSV(string(log.Category)),
				escapeCSV(log.UserIP),
				escapeCSV(log.UserAgent),
				escapeCSV(log.TargetType),
				escapeCSV(log.TargetID),
				escapeCSV(log.TargetName),
				escapeCSV(log.Details),
				escapeCSV(log.Status),
				escapeCSV(log.ErrorMessage),
			)
			c.Writer.WriteString(line)
		}
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))
		c.JSON(http.StatusOK, logs)
	}
}

// GetAuditLogStats returns statistics about audit logs
func (s *AppState) GetAuditLogStats(c *gin.Context) {
	db := dbWriter.GetDB()

	// Get counts by category
	categoryStats := make(map[string]int64)
	rows, err := db.Query(`
		SELECT category, COUNT(*) 
		FROM audit_logs 
		GROUP BY category
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int64
			if rows.Scan(&category, &count) == nil {
				categoryStats[category] = count
			}
		}
	}

	// Get counts by action (top 10)
	actionStats := make(map[string]int64)
	rows, err = db.Query(`
		SELECT action, COUNT(*) 
		FROM audit_logs 
		GROUP BY action 
		ORDER BY COUNT(*) DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var action string
			var count int64
			if rows.Scan(&action, &count) == nil {
				actionStats[action] = count
			}
		}
	}

	// Get total count
	var total int64
	db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&total)

	// Get today's count
	var today int64
	todayStart := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE timestamp >= ?", todayStart).Scan(&today)

	// Get error count
	var errors int64
	db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE status = 'error'").Scan(&errors)

	// Get oldest log timestamp
	var oldestTimestamp string
	db.QueryRow("SELECT MIN(timestamp) FROM audit_logs").Scan(&oldestTimestamp)

	c.JSON(http.StatusOK, gin.H{
		"total":            total,
		"today":            today,
		"errors":           errors,
		"by_category":      categoryStats,
		"top_actions":      actionStats,
		"oldest_timestamp": oldestTimestamp,
	})
}

// GetAuditLogSettings returns the current audit log settings
func (s *AppState) GetAuditLogSettings(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	settings := s.Config.AuditLogSettings
	if settings == nil {
		settings = &AuditLogSettings{
			RetentionDays: 30,
			Enabled:       true,
		}
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateAuditLogSettings updates the audit log settings
func (s *AppState) UpdateAuditLogSettings(c *gin.Context) {
	var settings AuditLogSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate retention days
	if settings.RetentionDays < 1 {
		settings.RetentionDays = 1
	}
	if settings.RetentionDays > 365 {
		settings.RetentionDays = 365
	}

	s.ConfigMu.Lock()
	s.Config.AuditLogSettings = &settings
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionSettingsUpdate, AuditCategorySettings,
		"settings", "audit_log", "Audit Log Settings",
		fmt.Sprintf("Updated retention to %d days, enabled: %v", settings.RetentionDays, settings.Enabled))

	c.JSON(http.StatusOK, settings)
}

// CleanupAuditLogs removes old audit logs based on retention settings
func CleanupAuditLogs(db *sql.DB, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format(time.RFC3339)
	_, err := db.Exec("DELETE FROM audit_logs WHERE timestamp < ?", cutoff)
	return err
}

// escapeCSV escapes a string for CSV output
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
