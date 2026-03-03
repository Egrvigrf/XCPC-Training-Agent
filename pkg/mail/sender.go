package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool // implicit TLS (e.g. 465)
}

func LoadFromEnv() (SMTPConfig, error) {
	get := func(k string) string { return strings.TrimSpace(os.Getenv(k)) }

	host := get("MP_SMTP_HOST")
	portStr := get("MP_SMTP_PORT")
	user := get("MP_SMTP_USER")
	pass := get("MP_SMTP_PASS")
	from := get("MP_SMTP_FROM")
	tlsStr := strings.ToLower(get("MP_SMTP_TLS"))

	if host == "" || portStr == "" || from == "" {
		return SMTPConfig{}, fmt.Errorf("smtp env not set: MP_SMTP_HOST/MP_SMTP_PORT/MP_SMTP_FROM required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("invalid MP_SMTP_PORT: %w", err)
	}

	return SMTPConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		From:     from,
		UseTLS:   tlsStr == "1" || tlsStr == "true" || tlsStr == "yes",
	}, nil
}

// Send sends a plain-text email. It only sends; it does not build business content.
func Send(ctx context.Context, cfg SMTPConfig, to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("empty recipients")
	}
	if subject == "" {
		subject = "(no subject)"
	}

	msg := buildMessage(cfg.From, to, subject, body)

	// context timeout safeguard
	if _, has := ctx.Deadline(); !has {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.UseTLS {
		return sendTLS(ctx, cfg, addr, to, msg)
	}
	return sendPlain(ctx, cfg, addr, to, msg)
}

func buildMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ",") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func sendPlain(ctx context.Context, cfg SMTPConfig, addr string, to []string, msg []byte) error {
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	type res struct{ err error }
	ch := make(chan res, 1)
	go func() {
		ch <- res{err: smtp.SendMail(addr, auth, cfg.From, to, msg)}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		return r.err
	}
}

func sendTLS(ctx context.Context, cfg SMTPConfig, addr string, to []string, msg []byte) error {
	d := &net.Dialer{}
	type res struct{ err error }
	ch := make(chan res, 1)

	go func() {
		conn, err := tls.DialWithDialer(d, "tcp", addr, &tls.Config{
			ServerName: cfg.Host,
		})
		if err != nil {
			ch <- res{err: err}
			return
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			ch <- res{err: err}
			return
		}
		defer c.Close()

		if cfg.Username != "" || cfg.Password != "" {
			auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
			if err := c.Auth(auth); err != nil {
				ch <- res{err: err}
				return
			}
		}

		if err := c.Mail(cfg.From); err != nil {
			ch <- res{err: err}
			return
		}
		for _, rcpt := range to {
			if err := c.Rcpt(rcpt); err != nil {
				ch <- res{err: err}
				return
			}
		}

		w, err := c.Data()
		if err != nil {
			ch <- res{err: err}
			return
		}
		if _, err := w.Write(msg); err != nil {
			_ = w.Close()
			ch <- res{err: err}
			return
		}
		_ = w.Close()

		ch <- res{err: c.Quit()}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		return r.err
	}
}
