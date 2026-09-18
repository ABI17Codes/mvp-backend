package services

import (
	"backend/models"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SMTPConfig holds the SMTP server connection settings
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// GetSMTPConfig loads SMTP settings from environment variables
func GetSMTPConfig() SMTPConfig {
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}

	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	if from == "" {
		from = user
	}

	return SMTPConfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     port,
		Username: user,
		Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		From:     from,
	}
}

// GetSuperAdminEmail retrieves the configured Super Admin email recipient
func GetSuperAdminEmail() string {
	email := strings.TrimSpace(os.Getenv("SUPER_ADMIN_EMAIL"))
	if email == "" {
		email = strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	}
	return email
}

// SendHTMLEmail sends an email with HTML content over SMTP (supporting STARTTLS and SSL/TLS)
func SendHTMLEmail(to string, subject string, htmlBody string) error {
	cfg := GetSMTPConfig()

	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		log.Println("⚠️  [Mail] SMTP credentials are not fully set in .env (SMTP_HOST, SMTP_USER, SMTP_PASSWORD). Skipping email.")
		return nil
	}

	if to == "" {
		log.Println("⚠️  [Mail] Destination email address is empty. Skipping email.")
		return nil
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	fromDisplay := "KadaiTheru Alert"
	fromHeader := fmt.Sprintf("%s <%s>", fromDisplay, cfg.From)

	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         cfg.Host,
	}

	// Port 465 uses direct SSL/TLS
	if cfg.Port == "465" {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("SMTP client initialization failed: %w", err)
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}

		if err = client.Mail(cfg.From); err != nil {
			return fmt.Errorf("SMTP MAIL command failed: %w", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT command failed: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA command failed: %w", err)
		}
		if _, err = w.Write([]byte(message.String())); err != nil {
			return fmt.Errorf("SMTP write body failed: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("SMTP close writer failed: %w", err)
		}

		return nil
	}

	// Port 587 and others use STARTTLS
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("TCP dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP client initialization failed: %w", err)
	}
	defer client.Quit()

	if hasStartTLS, _ := client.Extension("STARTTLS"); hasStartTLS {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS handshake failed: %w", err)
		}
	}

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err = client.Mail(cfg.From); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT command failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err = w.Write([]byte(message.String())); err != nil {
		return fmt.Errorf("SMTP write body failed: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close writer failed: %w", err)
	}

	return nil
}

// SendNewUserAlert asynchronously notifies the Super Admin of a new user event (Registration or First Login)
func SendNewUserAlert(eventType string, user models.User, clientIP string, userAgent string) {
	// Execute asynchronously in a background goroutine so API responses are never delayed
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("❌ [Mail] Panic recovered in SendNewUserAlert: %v\n", r)
			}
		}()

		superAdmin := GetSuperAdminEmail()
		if superAdmin == "" {
			log.Println("⚠️  [Mail] No Super Admin email found in SUPER_ADMIN_EMAIL or ADMIN_EMAIL.")
			return
		}

		frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		adminUsersURL := fmt.Sprintf("%s/admin/users", strings.TrimRight(frontendURL, "/"))

		nowFormatted := time.Now().Format("02 Jan 2006, 03:04:05 PM MST")
		if clientIP == "" {
			clientIP = "N/A"
		}
		if userAgent == "" {
			userAgent = "Unknown Client"
		}

		subject := fmt.Sprintf("🔔 [Super Admin Alert] %s: %s", eventType, user.Email)

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f3f4f6; margin: 0; padding: 24px; color: #1f2937; }
    .container { max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 16px rgba(0,0,0,0.06); border: 1px solid #e5e7eb; }
    .header { background: linear-gradient(135deg, #4f46e5 0%%, #7c3aed 100%%); color: #ffffff; padding: 24px; text-align: left; }
    .header h1 { margin: 0; font-size: 20px; font-weight: 700; letter-spacing: -0.02em; }
    .header p { margin: 6px 0 0 0; font-size: 13px; opacity: 0.85; }
    .content { padding: 24px; }
    .badge { display: inline-block; padding: 4px 10px; font-size: 12px; font-weight: 600; border-radius: 20px; background: #e0e7ff; color: #4338ca; margin-bottom: 16px; }
    .card { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; margin-bottom: 20px; }
    .row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f0f1f3; font-size: 14px; }
    .row:last-child { border-bottom: none; }
    .label { color: #6b7280; font-weight: 500; min-width: 120px; }
    .value { color: #111827; font-weight: 600; text-align: right; word-break: break-all; }
    .btn { display: inline-block; background: #4f46e5; color: #ffffff !important; text-decoration: none; padding: 10px 20px; border-radius: 6px; font-size: 14px; font-weight: 600; margin-top: 10px; }
    .footer { text-align: center; padding: 18px; font-size: 12px; color: #9ca3af; border-top: 1px solid #e5e7eb; background: #fafafa; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>KadaiTheru Platform Alert</h1>
      <p>Super Admin Notification System</p>
    </div>
    <div class="content">
      <div class="badge">🔔 %s</div>
      <p style="font-size: 15px; margin-top: 0; line-height: 1.5;">
        A new user activity has been recorded on the platform:
      </p>
      
      <div class="card">
        <div class="row">
          <span class="label">User Name:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">User Email:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">Assigned Role:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">User ID:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">Timestamp:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">IP Address:</span>
          <span class="value">%s</span>
        </div>
        <div class="row">
          <span class="label">Client Agent:</span>
          <span class="value" style="font-size: 12px;">%s</span>
        </div>
      </div>

      <div style="text-align: center; margin-top: 16px;">
        <a href="%s" class="btn" target="_blank">View User in Admin Panel</a>
      </div>
    </div>
    <div class="footer">
      This is an automated notification intended solely for the Super Admin.<br/>
      KadaiTheru SaaS Platform
    </div>
  </div>
</body>
</html>`,
			eventType,
			user.Name,
			user.Email,
			user.Role,
			user.ID.String(),
			nowFormatted,
			clientIP,
			userAgent,
			adminUsersURL,
		)

		log.Printf("📧 [Mail] Dispatching Super Admin alert to %s for user %s (%s)...\n", superAdmin, user.Email, eventType)
		if err := SendHTMLEmail(superAdmin, subject, htmlBody); err != nil {
			log.Printf("❌ [Mail] Failed to send email to Super Admin (%s): %v\n", superAdmin, err)
		} else {
			log.Printf("✅ [Mail] Successfully sent Super Admin alert to %s\n", superAdmin)
		}
	}()
}
