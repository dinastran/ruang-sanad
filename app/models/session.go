package models

import "time"

// Session represents a user session stored in database
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Data      string    `json:"data"` // JSON encoded session data
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the table name for Session
func (Session) TableName() string {
	return "sessions"
}

// SessionData represents the decoded session data
type SessionData struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}
