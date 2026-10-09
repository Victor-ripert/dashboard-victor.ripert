package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Used when the caller's context has no deadline
const defaultTimeout = 10 * time.Second

type SMTPConfig struct {
	Host string
	Port string
	// Sender, e.g. "Dashboard <noreply@dashboard.local>"
	From string
	// Optional. Only sent over an encrypted connection (STARTTLS) or to localhost.
	Username string
	Password string
}

// SMTPMailer sends emails to any standard SMTP server (Mailpit, Brevo, SendGrid...).
type SMTPMailer struct {
	addr string
	host string
	from *mail.Address
	auth smtp.Auth
}

func NewSMTPMailer(cfg SMTPConfig) (*SMTPMailer, error) {
	if cfg.Host == "" || cfg.Port == "" {
		return nil, errors.New("mailer: SMTP host and port are required")
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("mailer: invalid sender address %q: %w", cfg.From, err)
	}

	m := &SMTPMailer{addr: net.JoinHostPort(cfg.Host, cfg.Port), host: cfg.Host, from: from}
	if cfg.Username != "" {
		m.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return m, nil
}

// Addr returns the "host:port" of the SMTP server.
func (m *SMTPMailer) Addr() string {
	return m.addr
}

func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("mailer: invalid recipient %q: %w", msg.To, err)
	}
	data, err := buildMessage(m.from, to, msg.Subject, msg.Body, time.Now())
	if err != nil {
		return err
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return fmt.Errorf("mailer: connect to %s: %w", m.addr, err)
	}
	// Abort the conversation as soon as the context ends (timeout or cancellation)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mailer: %w", err)
	}
	defer client.Close()

	if err := m.deliver(client, to.Address, data); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("mailer: %w", ctx.Err())
		}
		return fmt.Errorf("mailer: %w", err)
	}
	return nil
}

// deliver runs the SMTP conversation on an open connection.
func (m *SMTPMailer) deliver(client *smtp.Client, to string, data []byte) error {
	// Encrypt the connection when the server supports it (real providers do, Mailpit doesn't by default)
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return err
		}
	}
	if m.auth != nil {
		if err := client.Auth(m.auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// buildMessage returns the raw email: headers, a blank line, then the body.
func buildMessage(from, to *mail.Address, subject, body string, date time.Time) ([]byte, error) {
	// A line break in a header would let the caller inject extra headers
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errors.New("mailer: subject must be a single line")
	}

	var buf bytes.Buffer
	header := func(key, value string) {
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}
	header("From", from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", subject))
	header("Date", date.Format(time.RFC1123Z))
	header("Message-ID", messageID(from.Address))
	header("MIME-Version", "1.0")
	header("Content-Type", `text/plain; charset="utf-8"`)
	header("Content-Transfer-Encoding", "quoted-printable")
	buf.WriteString("\r\n")

	// Quoted-printable keeps accents and long lines safe for SMTP
	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func messageID(sender string) string {
	random := make([]byte, 16)
	rand.Read(random)
	domain := sender[strings.LastIndex(sender, "@")+1:]
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(random), domain)
}
