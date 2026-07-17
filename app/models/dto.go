package models

// DTOs for request/response

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type UserResponse struct {
	ID            int64    `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Avatar        string   `json:"avatar"`
	Role          UserRole `json:"role"`
	EmailVerified bool     `json:"email_verified"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Avatar:        u.Avatar,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
	}
}
