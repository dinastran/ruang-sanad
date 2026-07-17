package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// MailerService handles email sending
type MailerService struct {
	querier   *queries.Querier
	appURL    string
	smtpHost  string
	smtpPort  int
	smtpUser  string
	smtpPass  string
	fromEmail string
	fromName  string
}

type ResetTokenEntry struct {
	UserID    int64
	Email     string
	Token     string
	ExpiresAt time.Time
}

func NewMailerService(querier *queries.Querier, smtpHost string, smtpPort int, smtpUser, smtpPass, fromEmail, fromName, appURL string) *MailerService {
	return &MailerService{
		querier:   querier,
		appURL:    appURL,
		smtpHost:  smtpHost,
		smtpPort:  smtpPort,
		smtpUser:  smtpUser,
		smtpPass:  smtpPass,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// SendPasswordResetEmail generates a token, stores it in DB, and sends the reset email
func (m *MailerService) SendPasswordResetEmail(ctx context.Context, email string, userID int64) error {
	// Generate reset token
	token, err := generateResetToken()
	if err != nil {
		return err
	}

	// Store token in database
	if err := m.querier.CreatePasswordReset(ctx, token, userID, email, time.Now().Add(1*time.Hour)); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Build reset URL with real token
	resetURL := fmt.Sprintf("%s/reset-password/%s", m.appURL, token)

	// Build email
	subject := "Reset Your Password"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #f97316, #ea580c); padding: 30px; text-align: center; border-radius: 12px 12px 0 0; }
        .header h1 { color: white; margin: 0; font-size: 24px; }
        .content { background: #f9fafb; padding: 30px; border-radius: 0 0 12px 12px; }
        .button { display: inline-block; background: #f97316; color: white; padding: 12px 30px; text-decoration: none; border-radius: 8px; font-weight: 600; margin: 20px 0; }
        .button:hover { background: #ea580c; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
        .warning { background: #fef3c7; border-left: 4px solid #f59e0b; padding: 15px; margin: 20px 0; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hi there,</p>
            <p>We received a request to reset your password. Click the button below to reset it:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #f97316;">%s</p>
            <div class="warning">
                <strong>⚠️ Important:</strong> This link will expire in 1 hour.
            </div>
            <p>If you didn't request this, you can safely ignore this email. Your password will remain unchanged.</p>
            <p>Best regards,<br>The Laju Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2026 Laju. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, resetURL, resetURL)

	// Send email
	return m.SendEmail(email, subject, body)
}

// SendEmail sends an email using SMTP
func (m *MailerService) SendEmail(to, subject, body string) error {
	// Build email headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", m.fromName, m.fromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""
	headers["Content-Transfer-Encoding"] = "quoted-printable"

	// Build message
	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n" + body)

	// Send email
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpHost)
	addr := fmt.Sprintf("%s:%d", m.smtpHost, m.smtpPort)

	return smtp.SendMail(addr, auth, m.fromEmail, []string{to}, []byte(message.String()))
}

// ValidateResetToken validates a reset token against the database
func (m *MailerService) ValidateResetToken(ctx context.Context, token string) (*ResetTokenEntry, error) {
	pr, err := m.querier.GetPasswordReset(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	return &ResetTokenEntry{
		UserID:    pr.UserID,
		Email:     pr.Email,
		Token:     pr.Token,
		ExpiresAt: pr.ExpiresAt,
	}, nil
}

// InvalidateResetToken marks a reset token as used
func (m *MailerService) InvalidateResetToken(ctx context.Context, token string) {
	if err := m.querier.MarkPasswordResetUsed(ctx, token); err != nil {
		slog.Error("failed to invalidate reset token", "error", err, "token", token)
	}
}

// generateResetToken generates a secure random token
func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CleanupExpiredTokens removes expired tokens from the database
// Call this periodically via a cron job or background goroutine
func (m *MailerService) CleanupExpiredTokens(ctx context.Context) {
	err := m.querier.DeleteExpiredPasswordResets(ctx)
	if err != nil {
		slog.Error("failed to cleanup expired tokens", "error", err)
	}
}
