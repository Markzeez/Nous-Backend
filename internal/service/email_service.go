package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"time"

	"medcon/internal/config"
)

// EmailService handles sending emails via SMTP
type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// IsConfigured returns true if SMTP is properly configured
func (e *EmailService) IsConfigured() bool {
	return e.cfg.SMTPUser != "" && e.cfg.SMTPPass != "" && e.cfg.SMTPHost != ""
}

// SendPasswordResetEmail sends a password reset email with a secure token
func (e *EmailService) SendPasswordResetEmail(toEmail, resetToken string) error {
	if !e.IsConfigured() {
		// Log the token for development purposes
		fmt.Printf("[DEV] Password reset token for %s: %s\n", toEmail, resetToken)
		fmt.Printf("[DEV] Reset link: %s/auth/reset-password?token=%s\n", e.cfg.FrontendURL, resetToken)
		return nil
	}

	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", e.cfg.FrontendURL, resetToken)

	subject := "MedConnect - Password Reset Request"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #0ea5e9, #06b6d4); padding: 30px; text-align: center; border-radius: 12px 12px 0 0; }
        .header h1 { color: white; margin: 0; font-size: 24px; }
        .content { background: #f8fafc; padding: 30px; border-radius: 0 0 12px 12px; }
        .button { display: inline-block; background: #0ea5e9; color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: 600; margin: 20px 0; }
        .button:hover { background: #0284c7; }
        .token { background: #e0f2fe; border: 1px solid #0ea5e9; padding: 15px; border-radius: 8px; font-family: monospace; word-break: break-all; margin: 15px 0; }
        .footer { text-align: center; margin-top: 20px; color: #64748b; font-size: 14px; }
        .warning { background: #fef3c7; border: 1px solid #f59e0b; padding: 15px; border-radius: 8px; margin: 15px 0; color: #92400e; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MedConnect</h1>
        </div>
        <div class="content">
            <h2>Password Reset Request</h2>
            <p>We received a request to reset your password. Click the button below to create a new password:</p>

            <div style="text-align: center;">
                <a href="%s" class="button">Reset My Password</a>
            </div>

            <p>Or copy this link into your browser:</p>
            <div class="token">%s</div>

            <div class="warning">
                <strong>⚠️ Security Notice:</strong>
                <ul>
                    <li>This link expires in <strong>1 hour</strong></li>
                    <li>If you didn't request this, please ignore this email</li>
                    <li>Never share this link with anyone</li>
                </ul>
            </div>

            <p>If you're having trouble with the button, you can also use the reset token directly:</p>
            <div class="token"><strong>Reset Token:</strong> %s</div>
        </div>
        <div class="footer">
            <p>This email was sent by MedConnect. If you didn't request this, no action is needed.</p>
            <p>&copy; 2024 MedConnect. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, resetLink, resetLink, resetToken)

	return e.sendEmail(toEmail, subject, body)
}

// SendDoctorAvailableNotification notifies a patient when a doctor becomes available
func (e *EmailService) SendDoctorAvailableNotification(toEmail, patientName, doctorName, specialty string) error {
	if !e.IsConfigured() {
		fmt.Printf("[DEV] Doctor available notification for %s: Dr. %s (%s) is now available\n", patientName, doctorName, specialty)
		return nil
	}

	subject := "MedConnect - A Doctor is Now Available for You"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #10b981, #059669); padding: 30px; text-align: center; border-radius: 12px 12px 0 0; }
        .header h1 { color: white; margin: 0; font-size: 24px; }
        .content { background: #f8fafc; padding: 30px; border-radius: 0 0 12px 12px; }
        .button { display: inline-block; background: #10b981; color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: 600; margin: 20px 0; }
        .doctor-card { background: white; border: 1px solid #e5e7eb; border-radius: 8px; padding: 20px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #64748b; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MedConnect</h1>
        </div>
        <div class="content">
            <h2>Good news, %s!</h2>
            <p>A <strong>%s specialist</strong> is now available to help you.</p>

            <div class="doctor-card">
                <h3 style="margin: 0 0 10px 0;">Dr. %s</h3>
                <p style="margin: 5px 0; color: #64748b;">Specialty: %s</p>
                <p style="margin: 5px 0; color: #64748b;">Status: Available now</p>
            </div>

            <div style="text-align: center;">
                <a href="%s/booking?type=CONSULTATION&specialty=%s" class="button">Book Now</a>
            </div>

            <p>Book quickly — availability can change!</p>
        </div>
        <div class="footer">
            <p>This email was sent by MedConnect.</p>
            <p>&copy; 2024 MedConnect. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, patientName, specialty, doctorName, specialty, e.cfg.FrontendURL, specialty)

	return e.sendEmail(toEmail, subject, body)
}

// sendEmail sends an email using SMTP
func (e *EmailService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", e.cfg.SMTPUser, e.cfg.SMTPPass, e.cfg.SMTPHost)

	// Construct message
	msg := fmt.Sprintf("From: %s\r\n"+ // From
		"To: %s\r\n"+ // To
		"Subject: %s\r\n"+ // Subject
		"MIME-Version: 1.0\r\n"+ // MIME
		"Content-Type: text/html; charset=UTF-8\r\n"+ // HTML
		"\r\n"+ // Blank line
		"%s", e.cfg.SMTPFrom, to, subject, body)

	// Send
	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)
	return smtp.SendMail(addr, auth, e.cfg.SMTPFrom, []string{to}, []byte(msg))
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// PasswordResetTokenStore is an in-memory store for password reset tokens
// In production, use Redis or a database table with TTL
type PasswordResetTokenStore struct {
	tokens map[string]TokenInfo
}

type TokenInfo struct {
	Email     string
	ExpiresAt time.Time
}

func NewPasswordResetTokenStore() *PasswordResetTokenStore {
	store := &PasswordResetTokenStore{
		tokens: make(map[string]TokenInfo),
	}
	// Cleanup expired tokens every hour
	go store.cleanup()
	return store
}

func (s *PasswordResetTokenStore) Store(email, token string) {
	s.tokens[token] = TokenInfo{
		Email:     email,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
}

func (s *PasswordResetTokenStore) Validate(token string) (string, bool) {
	info, ok := s.tokens[token]
	if !ok {
		return "", false
	}
	if time.Now().After(info.ExpiresAt) {
		delete(s.tokens, token)
		return "", false
	}
	return info.Email, true
}

func (s *PasswordResetTokenStore) Consume(token string) (string, bool) {
	email, ok := s.Validate(token)
	if ok {
		delete(s.tokens, token)
	}
	return email, ok
}

func (s *PasswordResetTokenStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		now := time.Now()
		for token, info := range s.tokens {
			if now.After(info.ExpiresAt) {
				delete(s.tokens, token)
			}
		}
	}
}
