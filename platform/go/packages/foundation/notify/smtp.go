package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig is the provider-agnostic configuration for an SMTP channel. Any
// transactional provider that speaks SMTP (Postmark, SES, Mailgun, a self-hosted
// relay) is configured by filling these fields — the adapter has no
// provider-specific code.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
	// TLS requests an authenticated submission. When false the adapter sends
	// without AUTH (an open relay / local MTA); when true it authenticates with
	// Username/Password via the standard PLAIN mechanism over the connection.
	TLS bool
}

// SMTPNotifier delivers a Message as a plain-text email via stdlib net/smtp.
// It is provider-agnostic: point SMTPConfig at any SMTP host.
type SMTPNotifier struct {
	cfg  SMTPConfig
	send smtpSendFunc
}

// smtpSendFunc matches smtp.SendMail so tests can inject a stub. The real
// adapter uses smtp.SendMail.
type smtpSendFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// NewSMTPNotifier builds an SMTP adapter from the given config.
func NewSMTPNotifier(cfg SMTPConfig) *SMTPNotifier {
	return &SMTPNotifier{cfg: cfg, send: smtp.SendMail}
}

// Notify renders the Message as a simple text email and sends it. The context
// is accepted for interface symmetry; net/smtp does not support cancellation.
func (s *SMTPNotifier) Notify(_ context.Context, msg Message) error {
	if s.cfg.From == "" || s.cfg.To == "" {
		return fmt.Errorf("smtp: From and To are required")
	}
	var auth smtp.Auth
	if s.cfg.TLS {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	if err := s.send(addr, auth, s.cfg.From, []string{s.cfg.To}, buildEmail(s.cfg, msg)); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}

// buildEmail assembles an RFC 5322 plain-text message (headers + body).
func buildEmail(cfg SMTPConfig, msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + cfg.From + "\r\n")
	b.WriteString("To: " + cfg.To + "\r\n")
	b.WriteString("Subject: " + msg.Title + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	if msg.Body != "" {
		b.WriteString(msg.Body + "\r\n")
	}
	for k, v := range msg.Fields {
		b.WriteString(k + ": " + v + "\r\n")
	}
	return []byte(b.String())
}
