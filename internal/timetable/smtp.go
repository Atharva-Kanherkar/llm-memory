package timetable

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
)

// EmailSender sends reminder emails.
type EmailSender interface {
	Available() bool
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPSender sends email directly using Go's net/smtp.
type SMTPSender struct {
	host string
	port int
	user string
	pass string
	from string
}

// NewSMTPSender creates an SMTP sender from timetable config.
func NewSMTPSender(cfg config.TimetableConfig) *SMTPSender {
	return &SMTPSender{
		host: strings.TrimSpace(cfg.SMTPHost),
		port: cfg.SMTPPort,
		user: strings.TrimSpace(cfg.SMTPUser),
		pass: cfg.SMTPPass,
		from: strings.TrimSpace(cfg.EmailFrom),
	}
}

// Available returns true if SMTP config is complete.
func (s *SMTPSender) Available() bool {
	if s == nil {
		return false
	}
	if s.host == "" || s.port <= 0 || s.from == "" {
		return false
	}
	if s.user == "" || s.pass == "" {
		return false
	}
	return true
}

// Send sends an email reminder.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	if !s.Available() {
		return fmt.Errorf("smtp sender is not available")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("missing recipient email")
	}

	fromAddr, err := extractAddress(s.from)
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	toAddr, err := extractAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	headerFrom := s.from
	if headerFrom == "" {
		headerFrom = fromAddr
	}

	msg := buildEmailMessage(headerFrom, to, subject, body)
	addr := s.host + ":" + strconv.Itoa(s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	if err := smtp.SendMail(addr, auth, fromAddr, []string{toAddr}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}
	return nil
}

func extractAddress(raw string) (string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return parsed.Address, nil
}

func buildEmailMessage(from, to, subject, body string) string {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}
