package messaging

import (
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	gomail "gopkg.in/gomail.v2"
)

// smtpEmailSender implements repo.EmailSender using SMTP via gomail.
type smtpEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPEmailSender constructs a new SMTPEmailSender.
func NewSMTPEmailSender(host string, port int, username, password, from string) repo.EmailSender {
	// Default from to username if not provided
	if from == "" {
		from = username
	}
	return &smtpEmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// Send sends an email with the given subject and body to the recipient.
func (m *smtpEmailSender) Send(to string, subject string, body string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	d := gomail.NewDialer(m.host, m.port, m.username, m.password)
	if err := d.DialAndSend(msg); err != nil {
		return fmt.Errorf("SMTPEmailSender - Send - DialAndSend: %w", err)
	}
	return nil
}
