package services

import (
	"context"
	"errors"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type UserService struct {
	querier *queries.Querier
}

func NewUserService(querier *queries.Querier) *UserService {
	return &UserService{
		querier: querier,
	}
}

// GetProfile retrieves a user's profile directly from DB.
func (s *UserService) GetProfile(userID int64) (*models.UserResponse, error) {
	user, err := s.querier.GetUserByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

// GetProfileByEmail retrieves a user's profile by email
func (s *UserService) GetProfileByEmail(email string) (*models.User, error) {
	return s.querier.GetUserByEmail(context.Background(), email)
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(userID int64, hashedPassword string) error {
	return s.querier.UpdateUserPassword(context.Background(), userID, hashedPassword)
}

// UpdateAvatar updates a user's avatar URL
func (s *UserService) UpdateAvatar(userID int64, avatarURL string) error {
	return s.querier.UpdateUserAvatar(context.Background(), userID, avatarURL)
}

// UpdateProfile updates a user's profile
func (s *UserService) UpdateProfile(userID int64, req models.UpdateProfileRequest) (*models.UserResponse, error) {
	user, err := s.querier.GetUserByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := s.querier.UpdateUser(context.Background(), user); err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := s.querier.GetUserByID(context.Background(), userID)
	if err != nil {
		return err
	}

	// Verify old password - user must have a password
	if !user.Password.Valid {
		return errors.New("invalid current password")
	}

	if !CheckPassword(oldPassword, user.Password.String) {
		return errors.New("invalid current password")
	}

	// Hash new password
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.querier.UpdateUserPassword(context.Background(), userID, hashedPassword)
}

// DeleteAccount deletes a user's account
func (s *UserService) DeleteAccount(userID int64) error {
	return s.querier.DeleteUser(context.Background(), userID)
}

// IsAdmin checks if a user is an admin (direct DB query).
func (s *UserService) IsAdmin(userID int64) (bool, error) {
	user, err := s.querier.GetUserByID(context.Background(), userID)
	if err != nil {
		return false, err
	}

	return user.Role == models.RoleAdmin, nil
}

// ListUsers returns all users
func (s *UserService) ListUsers() ([]models.UserResponse, error) {
	users, err := s.querier.ListUsers(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]models.UserResponse, len(users))
	for i, u := range users {
		avatar := ""
		if u.Avatar.Valid {
			avatar = u.Avatar.String
		}
		result[i] = models.UserResponse{
			ID:            u.ID,
			Email:         u.Email,
			Name:          u.Name,
			Avatar:        avatar,
			Role:          models.UserRole(u.Role),
			EmailVerified: u.EmailVerified,
		}
	}
	return result, nil
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(id int64, role string) error {
	return s.querier.UpdateUserRole(context.Background(), queries.UpdateUserRoleParams{
		Role:      role,
		UpdatedAt: time.Now(),
		ID:        id,
	})
}

// GetUserRole returns a user's role string
func (s *UserService) GetUserRole(id int64) (string, error) {
	return s.querier.GetUserRole(context.Background(), id)
}
