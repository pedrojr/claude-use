package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const usageURL = "https://api.anthropic.com/api/oauth/usage"

// Limit is one usage window returned by the API (utilization is 0-100).
type Limit struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    *string `json:"resets_at"`
}

// ResetTime returns the parsed reset instant, or zero time if absent.
func (l *Limit) ResetTime() time.Time {
	if l == nil || l.ResetsAt == nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, *l.ResetsAt)
	if err != nil {
		return time.Time{}
	}
	return t
}

type Usage struct {
	FiveHour       *Limit `json:"five_hour"`
	SevenDay       *Limit `json:"seven_day"`
	SevenDayOpus   *Limit `json:"seven_day_opus"`
	SevenDaySonnet *Limit `json:"seven_day_sonnet"`
}

type credentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		ExpiresAt        int64  `json:"expiresAt"`
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
}

func credentialsPath() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, ".credentials.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".credentials.json")
}

// loadCredentials reads the OAuth token that Claude Code keeps up to date.
// The file is re-read on every refresh so a token renewed by Claude Code is picked up.
func loadCredentials() (*credentials, error) {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		return nil, usageError{key: "errNoCreds"}
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, usageError{key: "errBadCreds"}
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return nil, usageError{key: "errLogin"}
	}
	return &c, nil
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// fetchUsage calls the same endpoint used by Claude Code's /usage screen.
func fetchUsage() (*Usage, string, error) {
	c, err := loadCredentials()
	if err != nil {
		return nil, "", err
	}
	plan := c.ClaudeAiOauth.SubscriptionType

	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, plan, err
	}
	req.Header.Set("Authorization", "Bearer "+c.ClaudeAiOauth.AccessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "claude-use/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, plan, usageError{key: "errOffline"}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, plan, usageError{key: "errExpired"}
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, plan, usageError{key: "errRateLimit"}
	case resp.StatusCode != http.StatusOK:
		return nil, plan, usageError{key: "errHTTP", args: []any{resp.StatusCode}}
	}

	var u Usage
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, plan, usageError{key: "errBadReply"}
	}
	return &u, plan, nil
}

func planLabel(p string) string {
	switch p {
	case "team":
		return T("planTeam")
	case "pro":
		return "Pro"
	case "max":
		return "Max"
	case "enterprise":
		return "Enterprise"
	case "":
		return ""
	}
	return p
}

// resetText mimics "Resets in 4 h 8 min".
func resetText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Until(t)
	if d <= 0 {
		return T("resetting")
	}
	mins := int(d.Round(time.Minute).Minutes())
	if mins < 1 { // less than 30 s left: show "1 min", not "0 min"
		mins = 1
	}
	days, hours, m := mins/(24*60), (mins/60)%24, mins%60
	switch {
	case days > 0 && hours == 0:
		return T("resetsD", days)
	case days > 0:
		return T("resetsDH", days, hours)
	case hours > 0:
		return T("resetsHM", hours, m)
	default:
		return T("resetsM", m)
	}
}
