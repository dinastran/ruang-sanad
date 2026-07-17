package models

import (
	"database/sql"
	"time"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID            int64          `json:"id"`
	Email         string         `json:"email"`
	Name          string         `json:"name"`
	Avatar        string         `json:"avatar"`
	Password      sql.NullString `json:"-"` // Hashed password, never return in JSON (NULL for OAuth users)
	Role          UserRole       `json:"role"`
	GoogleID      sql.NullString `json:"-"` // OAuth provider ID (NULL for email/password users)
	EmailVerified bool           `json:"email_verified"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}
