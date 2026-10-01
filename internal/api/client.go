package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"cursor-cost-monitor/internal/model"
	"cursor-cost-monitor/internal/tracker"
)

const (
	BaseURL   = "https://cursor.com"
	UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

type Client struct {
	httpClient *http.Client
	cookie     string
	bearer     string
}

func NewClient(cookie, bearer string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
		cookie: cookie,
		bearer: bearer,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, bodyPayload any) ([]byte, error) {
	url := BaseURL + path
	var bodyReader io.Reader
	if bodyPayload != nil {
		jsonData, err := json.Marshal(bodyPayload)
		if err != nil {
			return nil, fmt.Errorf("error serializing request payload: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", BaseURL)
	req.Header.Set("Referer", BaseURL+"/settings")
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if bodyPayload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error on %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %w", path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication error (401 Unauthorized): session token expired or invalid")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("access forbidden (403 Forbidden) on %s: %s", path, string(respBytes))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d on %s: %s", resp.StatusCode, path, string(respBytes))
	}

	return respBytes, nil
}

// GetUsage retrieves basic request usage
func (c *Client) GetUsage(ctx context.Context) (*model.UsageResponse, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/api/usage", nil)
	if err != nil {
		return nil, err
	}
	var res model.UsageResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error parsing /api/usage: %w", err)
	}
	return &res, nil
}

// GetStripe retrieves membership and billing profile info
func (c *Client) GetStripe(ctx context.Context) (*model.StripeResponse, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/api/auth/stripe", nil)
	if err != nil {
		return nil, err
	}
	var res model.StripeResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error parsing /api/auth/stripe: %w", err)
	}
	return &res, nil
}

// GetHardLimit retrieves configured billing hard limit
func (c *Client) GetHardLimit(ctx context.Context) (*model.HardLimitResponse, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/api/dashboard/get-hard-limit", map[string]any{})
	if err != nil {
		return nil, err
	}
	var res model.HardLimitResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error parsing get-hard-limit: %w", err)
	}
	return &res, nil
}

// GetTeams retrieves user's teams
func (c *Client) GetTeams(ctx context.Context) (*model.TeamsResponse, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/api/dashboard/teams", map[string]any{})
	if err != nil {
		return nil, err
	}
	var res model.TeamsResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error parsing teams: %w", err)
	}
	return &res, nil
}

// GetTeamSpend retrieves team member spending details
func (c *Client) GetTeamSpend(ctx context.Context, teamID int64) (*model.TeamSpendResponse, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/api/dashboard/get-team-spend", map[string]any{
		"teamId": teamID,
	})
	if err != nil {
		return nil, err
	}
	var res model.TeamSpendResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error parsing get-team-spend: %w", err)
	}
	return &res, nil
}

// GetCurrentPeriodUsage retrieves detailed usage breakdown from DashboardService
func (c *Client) GetCurrentPeriodUsage(ctx context.Context) (*model.PeriodUsageResponse, error) {
	url := "https://api2.cursor.sh/aiserver.v1.DashboardService/GetCurrentPeriodUsage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	var res model.PeriodUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FetchDashboard concurrently loads all relevant data and constructs an AggregatedDashboard
func (c *Client) FetchDashboard(ctx context.Context, currentUserEmail string) (*model.AggregatedDashboard, error) {
	var (
		wg         sync.WaitGroup
		usageRes   *model.UsageResponse
		stripeRes  *model.StripeResponse
		limitRes   *model.HardLimitResponse
		teamsRes   *model.TeamsResponse
		teamSpend  *model.TeamSpendResponse
		periodRes  *model.PeriodUsageResponse
		usageErr   error
		stripeErr  error
		limitErr   error
		teamsErr   error
		spendErr   error
		periodErr  error
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		usageRes, usageErr = c.GetUsage(ctx)
	}()

	go func() {
		defer wg.Done()
		stripeRes, stripeErr = c.GetStripe(ctx)
	}()

	go func() {
		defer wg.Done()
		limitRes, limitErr = c.GetHardLimit(ctx)
	}()

	go func() {
		defer wg.Done()
		teamsRes, teamsErr = c.GetTeams(ctx)
	}()

	go func() {
		defer wg.Done()
		periodRes, periodErr = c.GetCurrentPeriodUsage(ctx)
	}()

	wg.Wait()

	_ = limitErr
	_ = teamsErr
	_ = periodErr

	// If stripe failed, check if it's fatal
	if stripeErr != nil && usageErr != nil {
		return nil, fmt.Errorf("failed to fetch account data: %v", stripeErr)
	}

	var teamID int64
	var teamName string
	if stripeRes != nil && stripeRes.IsTeamMember && stripeRes.TeamID > 0 {
		teamID = stripeRes.TeamID
	}
	if teamsRes != nil && len(teamsRes.Teams) > 0 {
		if teamID == 0 {
			teamID = teamsRes.Teams[0].ID
		}
		for _, t := range teamsRes.Teams {
			if t.ID == teamID {
				teamName = t.Name
				break
			}
		}
	}

	var adminOnlyUsage bool
	if teamsRes != nil {
		for _, t := range teamsRes.Teams {
			if t.ID == teamID {
				adminOnlyUsage = t.AdminOnlyUsagePricing
				break
			}
		}
	}

	if teamID > 0 {
		teamSpend, spendErr = c.GetTeamSpend(ctx, teamID)
		_ = spendErr // Non-fatal if user is not authorized for team spend
	}

	dashboard := &model.AggregatedDashboard{
		UserEmail: currentUserEmail,
		FetchedAt: time.Now(),
	}

	if stripeRes != nil {
		dashboard.MembershipType = stripeRes.MembershipType
		dashboard.TeamID = stripeRes.TeamID
	}
	if teamName != "" {
		dashboard.TeamName = teamName
	}
	dashboard.AdminOnlyUsagePricing = adminOnlyUsage
	if limitRes != nil {
		dashboard.HardLimit = limitRes.HardLimit
	}

	if usageRes != nil && usageRes.Gpt4 != nil {
		dashboard.FastRequestsUsed = usageRes.Gpt4.NumRequests
		dashboard.TotalFastRequests = usageRes.Gpt4.NumRequestsTotal
		if usageRes.Gpt4.MaxRequestUsage != nil {
			dashboard.FastRequestsLimit = *usageRes.Gpt4.MaxRequestUsage
		} else {
			dashboard.FastRequestsLimit = 500
		}
	} else {
		dashboard.FastRequestsLimit = 500
	}

	if teamSpend != nil {
		dashboard.TeamMembers = teamSpend.TeamMemberSpend
		dashboard.CycleStart = model.ParseCycleTimestamp(teamSpend.SubscriptionCycleStart)
		dashboard.CycleEnd = model.ParseCycleTimestamp(teamSpend.NextCycleStart)

		if !dashboard.CycleEnd.IsZero() {
			rem := time.Until(dashboard.CycleEnd)
			days := int(rem.Hours() / 24)
			if days < 0 {
				days = 0
			}
			dashboard.DaysRemaining = days
		}

		// Find current user's spend
		foundUser := false
		lowerTarget := strings.ToLower(currentUserEmail)
		for _, m := range teamSpend.TeamMemberSpend {
			if lowerTarget != "" && (strings.EqualFold(m.Email, lowerTarget) || strings.Contains(strings.ToLower(m.Email), lowerTarget)) {
				dashboard.CurrentUserSpend = m
				foundUser = true
				break
			}
		}
		if !foundUser && len(teamSpend.TeamMemberSpend) > 0 {
			// Find member with highest activity or first
			for _, m := range teamSpend.TeamMemberSpend {
				if m.SpendCents > 0 || m.TotalPercentUsed > 0 {
					dashboard.CurrentUserSpend = m
					foundUser = true
					break
				}
			}
			if !foundUser {
				dashboard.CurrentUserSpend = teamSpend.TeamMemberSpend[0]
			}
		}
	}
	if periodRes != nil {
		dashboard.AutoModelsPercentUsed = periodRes.PlanUsage.AutoPercentUsed
		dashboard.NamedModelsPercentUsed = periodRes.PlanUsage.APIPercentUsed
		dashboard.IncludedPlanSpendUSD = float64(periodRes.PlanUsage.IncludedSpend) / 100.0
		dashboard.BonusSpendUSD = float64(periodRes.PlanUsage.BonusSpend) / 100.0
		dashboard.TeamPooledUsedUSD = float64(periodRes.SpendLimitUsage.PooledUsed) / 100.0

		pooledRem, _ := strconv.ParseInt(periodRes.SpendLimitUsage.PooledRemaining, 10, 64)
		dashboard.TeamPooledRemainingUSD = float64(pooledRem) / 100.0
		dashboard.AutoBucketModels = periodRes.AutoBucketModels

		if dashboard.TeamPooledUsedUSD > 0 && dashboard.CurrentUserSpend.SpendUSD() > 0 {
			others := dashboard.TeamPooledUsedUSD - dashboard.CurrentUserSpend.SpendUSD()
			if others > 0 {
				dashboard.TeamOthersSpendUSD = others
			}
		}
	}

	// Populate local per-model usage stats
	if modelStats, err := tracker.GetModelUsageStats(dashboard.CycleStart, dashboard.CurrentUserSpend.SpendUSD()); err == nil {
		dashboard.ModelStats = modelStats
	}

	return dashboard, nil
}
