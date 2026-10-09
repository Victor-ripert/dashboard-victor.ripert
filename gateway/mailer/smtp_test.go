package mailer

import (
	"context"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// fakeSMTPServer accepts one connection, answers like a minimal SMTP server
// and sends what it received on the returned channel.
type received struct {
	from, to string
	data     string
}

func fakeSMTPServer(t *testing.T) (host, port string, got <-chan received) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	ch := make(chan received, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		text := textproto.NewConn(conn)
		var r received
		text.PrintfLine("220 fake ready")
		for {
			line, err := text.ReadLine()
			if err != nil {
				return
			}
			switch command := strings.ToUpper(strings.SplitN(line, " ", 2)[0]); {
			case command == "EHLO" || command == "HELO":
				text.PrintfLine("250 hello")
			case strings.HasPrefix(strings.ToUpper(line), "MAIL FROM:"):
				r.from = line[len("MAIL FROM:"):]
				text.PrintfLine("250 ok")
			case strings.HasPrefix(strings.ToUpper(line), "RCPT TO:"):
				r.to = line[len("RCPT TO:"):]
				text.PrintfLine("250 ok")
			case command == "DATA":
				text.PrintfLine("354 go ahead")
				data, _ := text.ReadDotBytes()
				r.data = string(data)
				text.PrintfLine("250 queued")
			case command == "QUIT":
				text.PrintfLine("221 bye")
				ch <- r
				return
			default:
				text.PrintfLine("502 unknown command")
			}
		}
	}()

	host, port, _ = net.SplitHostPort(listener.Addr().String())
	return host, port, ch
}

func TestSendDeliversMessage(t *testing.T) {
	host, port, got := fakeSMTPServer(t)
	m, err := NewSMTPMailer(SMTPConfig{Host: host, Port: port, From: "Dashboard <noreply@dashboard.local>"})
	if err != nil {
		t.Fatal(err)
	}

	err = m.Send(context.Background(), Message{
		To:      "victor@example.com",
		Subject: "Votre code de vérification",
		Body:    "Your code is 123456",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	r := <-got
	if r.from != "<noreply@dashboard.local>" || r.to != "<victor@example.com>" {
		t.Errorf("envelope = %s -> %s", r.from, r.to)
	}
	msg, err := mail.ReadMessage(strings.NewReader(r.data))
	if err != nil {
		t.Fatalf("invalid email: %v", err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "Votre code de vérification" {
		t.Errorf("subject = %q", subject)
	}
	body, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
	// SMTP always ends the data with a line break
	if strings.TrimRight(string(body), "\r\n") != "Your code is 123456" {
		t.Errorf("body = %q", body)
	}
}

func TestSendTimesOutOnSilentServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			time.Sleep(2 * time.Second) // never greets the client
		}
	}()

	host, port, _ := net.SplitHostPort(listener.Addr().String())
	m, _ := NewSMTPMailer(SMTPConfig{Host: host, Port: port, From: "noreply@dashboard.local"})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = m.Send(ctx, Message{To: "victor@example.com", Subject: "Hi", Body: "Hi"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Send took %v, the context deadline was ignored", elapsed)
	}
}

func TestSendRejectsHeaderInjection(t *testing.T) {
	m, _ := NewSMTPMailer(SMTPConfig{Host: "127.0.0.1", Port: "1", From: "noreply@dashboard.local"})

	cases := []Message{
		{To: "victor@example.com\r\nBcc: attacker@example.com", Subject: "Hi"},
		{To: "victor@example.com", Subject: "Hi\r\nBcc: attacker@example.com"},
		{To: "not an email", Subject: "Hi"},
	}
	for _, msg := range cases {
		if err := m.Send(context.Background(), msg); err == nil {
			t.Errorf("Send(%q, %q) should fail", msg.To, msg.Subject)
		}
	}
}

func TestNewSMTPMailerValidatesConfig(t *testing.T) {
	cases := []SMTPConfig{
		{Port: "1025", From: "noreply@dashboard.local"},
		{Host: "mailpit", From: "noreply@dashboard.local"},
		{Host: "mailpit", Port: "1025", From: "not an email"},
	}
	for _, cfg := range cases {
		if _, err := NewSMTPMailer(cfg); err == nil {
			t.Errorf("NewSMTPMailer(%+v) should fail", cfg)
		}
	}

	m, err := NewSMTPMailer(SMTPConfig{Host: "mailpit", Port: "1025", From: "noreply@dashboard.local"})
	if err != nil || m.Addr() != "mailpit:1025" {
		t.Errorf("valid config: addr=%v err=%v", m.Addr(), err)
	}
}
