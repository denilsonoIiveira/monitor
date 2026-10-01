package config

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

type DetectedCredentials struct {
	AccessToken string
	AuthID      string
	Email       string
	Cookie      string
	Source      string
}

// FindCursorStateDB returns the standard path to Cursor's state.vscdb depending on the OS
func FindCursorStateDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	var candidates []string

	switch runtime.GOOS {
	case "linux":
		candidates = []string{
			filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "state.vscdb"),
			filepath.Join(home, ".config", "cursor", "User", "globalStorage", "state.vscdb"),
		}
	case "darwin":
		candidates = []string{
			filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb"),
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			candidates = []string{
				filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb"),
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

// DetectFromLocalCursor tries to read authentication credentials directly from local Cursor state.vscdb
func DetectFromLocalCursor() (*DetectedCredentials, error) {
	dbPath := FindCursorStateDB()
	if dbPath == "" {
		return nil, fmt.Errorf("local Cursor state database not found")
	}

	// Open read-only
	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(dbPath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer db.Close()

	var creds DetectedCredentials
	creds.Source = "local_cursor_state (" + dbPath + ")"

	keys := []string{
		"cursorAuth/accessToken",
		"cursorAuth/stripeMembershipAuthId",
		"cursorAuth/cachedEmail",
	}

	for _, k := range keys {
		var val string
		err := db.QueryRow("SELECT value FROM ItemTable WHERE key = ? LIMIT 1", k).Scan(&val)
		if err != nil {
			continue
		}
		switch k {
			case "cursorAuth/accessToken":
				creds.AccessToken = val
			case "cursorAuth/stripeMembershipAuthId":
				creds.AuthID = val
			case "cursorAuth/cachedEmail":
				creds.Email = val
		}
	}

	if creds.AccessToken == "" {
		return nil, fmt.Errorf("cursorAuth/accessToken not found in local state database")
	}

	// Format cookie: WorkosCursorSessionToken=<escapedAuthId>::<token>
	if creds.AuthID != "" {
		creds.Cookie = fmt.Sprintf("WorkosCursorSessionToken=%s%%3A%%3A%s", url.QueryEscape(creds.AuthID), creds.AccessToken)
	} else {
		creds.Cookie = fmt.Sprintf("WorkosCursorSessionToken=%s", creds.AccessToken)
	}

	return &creds, nil
}

// ResolveCredentials resolves credentials using env vars, config file or auto-detection
func ResolveCredentials(manualToken, manualAuthID, manualCookie string) (*DetectedCredentials, error) {
	// 1. If explicit cookie provided
	if manualCookie != "" {
		return &DetectedCredentials{
			Cookie: manualCookie,
			Source: "manual_cookie_flag",
		}, nil
	}

	// 2. If explicit token provided
	if manualToken != "" {
		cookie := manualToken
		if !strings.HasPrefix(cookie, "WorkosCursorSessionToken=") {
			if manualAuthID != "" {
				cookie = fmt.Sprintf("WorkosCursorSessionToken=%s%%3A%%3A%s", url.QueryEscape(manualAuthID), manualToken)
			} else {
				cookie = fmt.Sprintf("WorkosCursorSessionToken=%s", manualToken)
			}
		}
		return &DetectedCredentials{
			AccessToken: manualToken,
			AuthID:      manualAuthID,
			Cookie:      cookie,
			Source:      "manual_token_flag",
		}, nil
	}

	// 3. Env variables
	envCookie := os.Getenv("CURSOR_SESSION_TOKEN")
	envToken := os.Getenv("CURSOR_ACCESS_TOKEN")
	envAuthID := os.Getenv("CURSOR_AUTH_ID")

	if envCookie != "" {
		if !strings.HasPrefix(envCookie, "WorkosCursorSessionToken=") {
			envCookie = fmt.Sprintf("WorkosCursorSessionToken=%s", envCookie)
		}
		return &DetectedCredentials{
			Cookie: envCookie,
			Source: "env:CURSOR_SESSION_TOKEN",
		}, nil
	}

	if envToken != "" {
		var cookie string
		if envAuthID != "" {
			cookie = fmt.Sprintf("WorkosCursorSessionToken=%s%%3A%%3A%s", url.QueryEscape(envAuthID), envToken)
		} else {
			cookie = fmt.Sprintf("WorkosCursorSessionToken=%s", envToken)
		}
		return &DetectedCredentials{
			AccessToken: envToken,
			AuthID:      envAuthID,
			Cookie:      cookie,
			Source:      "env:CURSOR_ACCESS_TOKEN",
		}, nil
	}

	// 4. Auto-detect from local Cursor state
	detected, err := DetectFromLocalCursor()
	if err == nil && detected != nil {
		return detected, nil
	}

	return nil, fmt.Errorf("no authentication credentials found. Run 'curcost auth login' or specify --token / CURSOR_SESSION_TOKEN")
}
