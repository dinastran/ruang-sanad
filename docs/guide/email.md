# Email

This guide covers email configuration in Laju Go.

## Overview

Laju Go includes a mailer service for sending password reset emails via SMTP. Email sending is **inline** (no template files) — all HTML is generated as Go string literals.

## Configuration

### Environment Variables

```bash
# .env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
FROM_EMAIL=noreply@example.com
FROM_NAME=Laju Go
APP_URL=http://localhost:8080
```

## Mailer Service

### Implementation

```go
// app/services/mailer.go
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
```

### Sending Password Reset Email

The method generates a secure random token, stores it in DB, and sends an inline HTML email:

```go
func (m *MailerService) SendPasswordResetEmail(ctx context.Context, email string, userID int64) error {
    token, err := generateResetToken()
    if err != nil {
        return err
    }

    // Store token in database (password_resets table)
    if err := m.querier.CreatePasswordReset(ctx, token, userID, email, time.Now().Add(1*time.Hour)); err != nil {
        return fmt.Errorf("failed to store reset token: %w", err)
    }

    resetURL := fmt.Sprintf("%s/reset-password/%s", m.appURL, token)

    // Inline HTML — no template files
    subject := "Reset Your Password"
    body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>...</head>
<body>
    <h1>Password Reset</h1>
    <a href="%s">Reset your password</a>
    <p>This link expires in 1 hour.</p>
</body>
</html>`, resetURL)

    return m.SendEmail(email, subject, body)
}
```

### Sending Raw Email

```go
func (m *MailerService) SendEmail(to, subject, body string) error {
    headers := make(map[string]string)
    headers["From"] = fmt.Sprintf("%s <%s>", m.fromName, m.fromEmail)
    headers["To"] = to
    headers["Subject"] = subject
    headers["MIME-Version"] = "1.0"
    headers["Content-Type"] = "text/html; charset=\"utf-8\""

    var message strings.Builder
    for key, value := range headers {
        message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
    }
    message.WriteString("\r\n" + body)

    auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpHost)
    addr := fmt.Sprintf("%s:%d", m.smtpHost, m.smtpPort)
    return smtp.SendMail(addr, auth, m.fromEmail, []string{to}, []byte(message.String()))
}
```

### Token Validation

```go
func (m *MailerService) ValidateResetToken(ctx context.Context, token string) (*ResetTokenEntry, error) {
    pr, err := m.querier.GetPasswordReset(ctx, token)
    if err != nil {
        return nil, fmt.Errorf("invalid or expired token")
    }
    return &ResetTokenEntry{UserID: pr.UserID, Email: pr.Email, Token: pr.Token, ExpiresAt: pr.ExpiresAt}, nil
}
```

## Password Reset Flow

### Handler

```go
// app/handlers/password-reset.go
func (h *PasswordResetHandler) SendResetLink(c *fiber.Ctx) error {
    // Parse email
    // Don't reveal if email exists (security best practice)
    user, err := h.userService.GetProfileByEmail(req.Email)
    if err != nil {
        return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
            "success": "If an account exists, we've sent a reset link.",
        })
    }

    h.mailerService.SendPasswordResetEmail(c.Context(), user.Email, user.ID)
    return h.inertiaService.Render(c, "auth/ForgotPassword", fiber.Map{
        "success": "If an account exists, we've sent a reset link.",
    })
}
```

## Route Setup

```go
// routes/web.go
app.Get("/forgot-password", passwordResetHandler.ShowForgotPasswordForm)
app.Post("/forgot-password", passwordResetHandler.SendResetLink, middlewares.PasswordResetRateLimit.Limit())
app.Get("/reset-password/:token", passwordResetHandler.ShowResetPasswordForm)
app.Post("/reset-password/:token", passwordResetHandler.ResetPassword)
```

## SMTP Providers

### Gmail

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-16-character-app-password
```

> 🔐 Use an [App Password](https://support.google.com/accounts/answer/185833), not your regular Gmail password.

### SendGrid

```bash
SMTP_HOST=smtp.sendgrid.com
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-sendgrid-api-key
```

### Mailgun

```bash
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@yourdomain.mailgun.org
SMTP_PASS=your-mailgun-api-key
```

## Best Practices

1. **Don't reveal if email exists** — Always return generic success message
2. **Use goroutines for async** — Server responds faster when email is sent asynchronously
3. **Handle errors gracefully** — Log but don't expose to user
