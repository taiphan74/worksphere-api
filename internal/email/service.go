package email

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"worksphere-api/internal/config"
)

type Service interface {
	Send(ctx context.Context, to string, subject string, body string) error
	SendHTML(ctx context.Context, to string, subject string, html string) error
}

type smtpService struct {
	host string
	port int
	user string
	pass string
	from string
	addr string
	auth smtp.Auth
}

func NewSMTPService(cfg config.SMTPConfig) Service {
	return &smtpService{
		host: cfg.Host,
		port: cfg.Port,
		user: cfg.User,
		pass: cfg.Pass,
		from: cfg.From,
		addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		auth: smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host),
	}
}

func (s *smtpService) Send(ctx context.Context, to string, subject string, body string) error {
	return s.send(ctx, to, subject, body, "text/plain; charset=UTF-8")
}

func (s *smtpService) SendHTML(ctx context.Context, to string, subject string, html string) error {
	return s.send(ctx, to, subject, html, "text/html; charset=UTF-8")
}

func (s *smtpService) send(_ context.Context, to string, subject string, body string, contentType string) error {
	normalizedTo := strings.TrimSpace(to)
	normalizedSubject := strings.TrimSpace(subject)
	normalizedBody := strings.TrimSpace(body)

	if normalizedTo == "" {
		return fmt.Errorf("send email: recipient must not be empty")
	}

	if normalizedSubject == "" {
		return fmt.Errorf("send email: subject must not be empty")
	}

	if normalizedBody == "" {
		return fmt.Errorf("send email: body must not be empty")
	}

	if _, err := mail.ParseAddress(normalizedTo); err != nil {
		return fmt.Errorf("send email: invalid recipient address: %w", err)
	}

	message := buildMessage(s.from, normalizedTo, normalizedSubject, normalizedBody, contentType)

	if err := smtp.SendMail(s.addr, s.auth, s.from, []string{normalizedTo}, []byte(message)); err != nil {
		return fmt.Errorf("send email via smtp %s: %w", s.addr, err)
	}

	return nil
}

func buildMessage(from, to, subject, body, contentType string) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: %s", contentType),
		"",
		body,
	}, "\r\n")
}
