package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type PasswordResetHandler struct {
	mailerService  *services.MailerService
	userService    *services.UserService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewPasswordResetHandler(
	mailerService *services.MailerService,
	userService *services.UserService,
	store *session.Store,
	inertiaService *services.InertiaService,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		mailerService:  mailerService,
		userService:    userService,
		store:          store,
		inertiaService: inertiaService,
	}
}

// ShowForgotPasswordForm displays the forgot password page
func (h *PasswordResetHandler) ShowForgotPasswordForm(c *fiber.Ctx) error {
	return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
		"Title": "Forgot Password",
	})
}

// SendResetLink handles sending password reset link
func (h *PasswordResetHandler) SendResetLink(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate email
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Please provide a valid email address",
		})
	}

	successMsg := "If an account exists with that email, we've sent a password reset link."

	// Don't reveal whether email exists (security best practice)
	user, err := h.userService.GetProfileByEmail(req.Email)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
			"success": successMsg,
		})
	}

	// MailerService generates the token, stores it in DB, and sends the email
	if err := h.mailerService.SendPasswordResetEmail(c.Context(), user.Email, user.ID); err != nil {
		return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
			"success": successMsg,
		})
	}

	return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
		"success": successMsg,
	})
}

// ShowResetPasswordForm displays the reset password page
func (h *PasswordResetHandler) ShowResetPasswordForm(c *fiber.Ctx) error {
	token := c.Params("token")

	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid reset link",
		})
	}

	// Validate token
	_, err := h.mailerService.ValidateResetToken(c.Context(), token)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
			"error": "Invalid or expired reset link",
		})
	}

	return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
		"Title": "Reset Password",
		"token": token,
	})
}

// ResetPassword handles password reset
func (h *PasswordResetHandler) ResetPassword(c *fiber.Ctx) error {
	token := c.Params("token")

	var req struct {
		Password          string `json:"password"`
		PasswordConfirmed string `json:"password_confirmation"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate password
	if req.Password == "" || len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}

	if req.Password != req.PasswordConfirmed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Passwords do not match",
		})
	}

	// Validate token
	tokenEntry, err := h.mailerService.ValidateResetToken(c.Context(), token)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
			"token": token,
			"error": "Invalid or expired reset link",
		})
	}

	// Get user
	user, err := h.userService.GetProfile(tokenEntry.UserID)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
			"token": token,
			"error": "User not found",
		})
	}

	// Hash new password
	hashedPassword, err := services.HashPassword(req.Password)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
			"token": token,
			"error": "Failed to reset password",
		})
	}

	// Update user password
	err = h.userService.UpdatePassword(user.ID, hashedPassword)
	if err != nil {
		return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
			"token": token,
			"error": "Failed to reset password",
		})
	}

	// Invalidate token so it can't be reused
	h.mailerService.InvalidateResetToken(c.Context(), token)

	return h.inertiaService.Render(c, "auth/ResetPassword", fiber.Map{
		"token":   token,
		"success": "Password reset successfully. You can now login with your new password.",
	})
}
