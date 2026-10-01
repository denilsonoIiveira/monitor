package tracker

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"cursor-cost-monitor/internal/model"

	_ "modernc.org/sqlite"
)

// FindAITrackingDB returns the path to Cursor's ai-code-tracking.db
func FindAITrackingDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	var candidates []string
	switch runtime.GOOS {
	case "linux", "darwin":
		candidates = []string{
			filepath.Join(home, ".cursor", "ai-tracking", "ai-code-tracking.db"),
			filepath.Join(home, ".config", "Cursor", "ai-tracking", "ai-code-tracking.db"),
		}
	case "windows":
		appData := os.Getenv("USERPROFILE")
		if appData != "" {
			candidates = []string{
				filepath.Join(appData, ".cursor", "ai-tracking", "ai-code-tracking.db"),
			}
		}
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// GetModelUsageStats retrieves all models used within the cycle and calculates cost share
func GetModelUsageStats(cycleStart time.Time, totalSpendUSD float64) ([]model.ModelUsageStat, error) {
	dbPath := FindAITrackingDB()
	if dbPath == "" {
		return nil, fmt.Errorf("ai-code-tracking.db not found")
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(dbPath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open ai-tracking db: %w", err)
	}
	defer db.Close()

	cycleStartMs := cycleStart.UnixMilli()
	if cycleStart.IsZero() {
		// Default to last 30 days
		cycleStartMs = time.Now().AddDate(0, 0, -30).UnixMilli()
	}

	query := `
		SELECT 
			COALESCE(model, 'composer-default') as model_name,
			count(DISTINCT conversationId) as convs,
			count(DISTINCT requestId) as reqs,
			count(*) as blocks,
			max(createdAt) as last_use
		FROM ai_code_hashes
		WHERE createdAt >= ?
		GROUP BY model_name
		ORDER BY reqs DESC, blocks DESC
	`

	rows, err := db.Query(query, cycleStartMs)
	if err != nil {
		return nil, fmt.Errorf("failed to query model stats: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelUsageStat
	totalReqs := 0

	for rows.Next() {
		var (
			s        model.ModelUsageStat
			lastUnix int64
		)
		if err := rows.Scan(&s.Name, &s.Conversations, &s.RequestsCount, &s.CodeBlocks, &lastUnix); err != nil {
			continue
		}
		if s.RequestsCount == 0 {
			continue
		}
		if lastUnix > 0 {
			s.LastUsed = time.UnixMilli(lastUnix)
		}
		totalReqs += s.RequestsCount
		stats = append(stats, s)
	}

	// Calculate percentages and estimated cost allocation
	for i := range stats {
		if totalReqs > 0 {
			stats[i].PercentOfUsage = (float64(stats[i].RequestsCount) / float64(totalReqs)) * 100.0
			if totalSpendUSD > 0 {
				stats[i].EstimatedCost = (float64(stats[i].RequestsCount) / float64(totalReqs)) * totalSpendUSD
			}
		}
	}

	return stats, nil
}
