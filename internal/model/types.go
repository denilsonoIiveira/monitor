package model

import (
	"strconv"
	"time"
)

// UsageResponse represents the response from /api/usage
type UsageResponse struct {
	Gpt4         *ModelUsageItem `json:"gpt-4,omitempty"`
	StartOfMonth string          `json:"startOfMonth,omitempty"`
}

type ModelUsageItem struct {
	NumRequests      int  `json:"numRequests"`
	NumRequestsTotal int  `json:"numRequestsTotal"`
	NumTokens        int  `json:"numTokens"`
	MaxTokenUsage    *int `json:"maxTokenUsage"`
	MaxRequestUsage  *int `json:"maxRequestUsage"`
}

// StripeResponse represents /api/auth/stripe
type StripeResponse struct {
	MembershipType         string  `json:"membershipType"`
	PaymentID              string  `json:"paymentId"`
	CustomerBalance        float64 `json:"customerBalance"`
	IsTeamMember           bool    `json:"isTeamMember"`
	TeamID                 int64   `json:"teamId"`
	TeamMembershipType     string  `json:"teamMembershipType"`
	IndividualMembership   string  `json:"individualMembershipType"`
	IsYearlyPlan           bool    `json:"isYearlyPlan"`
}

// HardLimitResponse represents /api/dashboard/get-hard-limit
type HardLimitResponse struct {
	HardLimit float64 `json:"hardLimit"`
}

// TeamsResponse represents /api/dashboard/teams
type TeamsResponse struct {
	Teams []TeamItem `json:"teams"`
}

type TeamItem struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Role                string `json:"role"`
	Seats               int    `json:"seats"`
	HasBilling          bool   `json:"hasBilling"`
	RequestQuotaPerSeat int    `json:"requestQuotaPerSeat"`
	AdminOnlyUsagePricing bool `json:"adminOnlyUsagePricing"`
}

// TeamSpendResponse represents /api/dashboard/get-team-spend
type TeamSpendResponse struct {
	TeamMemberSpend        []MemberSpend `json:"teamMemberSpend"`
	SubscriptionCycleStart string        `json:"subscriptionCycleStart"`
	NextCycleStart         string        `json:"nextCycleStart"`
	TotalMembers           int           `json:"totalMembers"`
}

type MemberSpend struct {
	UserID             int64   `json:"userId"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	Role               string  `json:"role"`
	SpendCents         int64   `json:"spendCents"`
	IncludedSpendCents int64   `json:"includedSpendCents"`
	BillingTier        string  `json:"billingTier"`
	AutoPercentUsed    float64 `json:"autoPercentUsed"`
	APIPercentUsed     float64 `json:"apiPercentUsed"`
	TotalPercentUsed   float64 `json:"totalPercentUsed"`
}

func (m MemberSpend) SpendUSD() float64 {
	return float64(m.SpendCents) / 100.0
}

func (m MemberSpend) IncludedSpendUSD() float64 {
	return float64(m.IncludedSpendCents) / 100.0
}

// PeriodUsageResponse represents the response from DashboardService/GetCurrentPeriodUsage
type PeriodUsageResponse struct {
	PlanUsage struct {
		TotalSpend       int64   `json:"totalSpend"`
		IncludedSpend    int64   `json:"includedSpend"`
		BonusSpend       int64   `json:"bonusSpend"`
		Limit            int64   `json:"limit"`
		AutoPercentUsed  float64 `json:"autoPercentUsed"`
		APIPercentUsed   float64 `json:"apiPercentUsed"`
		TotalPercentUsed float64 `json:"totalPercentUsed"`
	} `json:"planUsage"`
	SpendLimitUsage struct {
		TotalSpend      int64  `json:"totalSpend"`
		PooledLimit     string `json:"pooledLimit"`
		PooledUsed      int64  `json:"pooledUsed"`
		PooledRemaining string `json:"pooledRemaining"`
		IndividualUsed  int64  `json:"individualUsed"`
	} `json:"spendLimitUsage"`
	AutoBucketModels []string `json:"autoBucketModels"`
}

// AggregatedDashboard represents unified data for TUI/CLI display
type AggregatedDashboard struct {
	UserEmail              string
	MembershipType         string
	TeamName               string
	TeamID                 int64
	AdminOnlyUsagePricing  bool
	HardLimit              float64
	CycleStart             time.Time
	CycleEnd               time.Time
	DaysRemaining          int
	CurrentUserSpend       MemberSpend
	TeamMembers            []MemberSpend
	FastRequestsUsed       int
	FastRequestsLimit      int
	TotalFastRequests      int
	AutoModelsPercentUsed  float64
	NamedModelsPercentUsed float64
	BonusSpendUSD          float64
	IncludedPlanSpendUSD   float64
	TeamPooledUsedUSD      float64
	TeamPooledRemainingUSD float64
	TeamOthersSpendUSD     float64
	AutoBucketModels       []string
	ModelStats             []ModelUsageStat
	FetchedAt              time.Time
}

type ModelUsageStat struct {
	Name           string    `json:"name"`
	RequestsCount  int       `json:"requestsCount"`
	Conversations  int       `json:"conversations"`
	CodeBlocks     int       `json:"codeBlocks"`
	PercentOfUsage float64   `json:"percentOfUsage"`
	EstimatedCost  float64   `json:"estimatedCost"`
	LastUsed       time.Time `json:"lastUsed"`
}

func ParseCycleTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err == nil && ms > 0 {
		return time.UnixMilli(ms)
	}
	// Try ISO string fallback
	t, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return t
	}
	return time.Time{}
}
