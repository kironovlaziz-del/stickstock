package delivery

import (
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// SendEmail sends a plain-text email via SMTP with PLAIN auth — covers
// the common case (Gmail, SES SMTP, Mailgun SMTP, etc.) without pulling
// in a third-party mail library.
func SendEmail(cfg SMTPConfig, to, subject, body string) error {
	if cfg.Host == "" {
		return fmt.Errorf("SMTP is not configured (SMTP_HOST is empty)")
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		cfg.From, to, subject, body,
	)

	addr := cfg.Host + ":" + cfg.Port
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
}
