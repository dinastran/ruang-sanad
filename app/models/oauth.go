package models

import "time"

// OAuthClient is an AI application (Claude.ai, ChatGPT, ...) registered via
// OAuth Dynamic Client Registration to connect to /mcp.
type OAuthClient struct {
	ClientID                string
	ClientSecretHash        string
	ClientName              string
	RedirectURIs            []string
	TokenEndpointAuthMethod string
	CreatedAt               time.Time
}

// OAuthAuthorizationCode is a one-time code issued after the user approves
// access on the consent page.
type OAuthAuthorizationCode struct {
	CodeHash      string
	ClientID      string
	UserID        int64
	RedirectURI   string
	CodeChallenge string
	CanWrite      bool
	ExpiresAt     time.Time
}

// OAuthTokenAuth is an OAuth access or refresh token joined with its
// connection (ai_tokens row) and owner.
type OAuthTokenAuth struct {
	TokenID        int64
	Jenis          string // access / refresh
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	GrantID        int64
	GrantClientID  string
	GrantExpiresAt time.Time
	GrantRevokedAt *time.Time
	LastUsedAt     *time.Time
	CanWrite       bool
	UserID         int64
	UserName       string
	Role           string
}
