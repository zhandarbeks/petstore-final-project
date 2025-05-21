package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendEmail(to []string, subject, body string, isHTML bool) error
}

// smtpEmailSender is an SMTP implementation of EmailSender.
type smtpEmailSender struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string // Usually the email address
	smtpPassword string // For Gmail, this should be an App Password
	senderEmail  string // The "From" address
}

// NewSMTPEmailSender creates a new SMTPEmailSender.
func NewSMTPEmailSender(host string, port int, username, password, senderEmail string) (EmailSender, error) {
	if host == "" || port == 0 || username == "" || password == "" || senderEmail == "" {
		return nil, fmt.Errorf("SMTP configuration (host, port, username, password, senderEmail) cannot be empty")
	}
	return &smtpEmailSender{
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
		senderEmail:  senderEmail,
	}, nil
}

// SendEmail sends an email using the configured SMTP server.
func (s *smtpEmailSender) SendEmail(to []string, subject, body string, isHTML bool) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	// Construct the email message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.senderEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))) // Comma-separated list for multiple recipients
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	if isHTML {
		msg.WriteString("MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\";\r\n")
	}
	msg.WriteString("\r\n") // Empty line separates headers from body
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)

	// Send the email
	// For TLS connections (common, e.g., port 587 with STARTTLS)
	// For SSL connections (less common, e.g., port 465), the approach is different.
	// This example assumes STARTTLS on port 587 or plain SMTP on port 25.
	// Gmail uses STARTTLS on port 587.

	var err error
	if s.smtpPort == 465 { // SSL/TLS direct connection
		tlsconfig := &tls.Config{
			ServerName: s.smtpHost,
		}
		conn, errDial := tls.Dial("tcp", addr, tlsconfig)
		if errDial != nil {
			log.Printf("Notification Service | SMTP Error (tls.Dial): %v", errDial)
			return fmt.Errorf("failed to dial SMTP server (SSL/TLS): %w", errDial)
		}
		defer conn.Close()

		client, errClient := smtp.NewClient(conn, s.smtpHost)
		if errClient != nil {
			log.Printf("Notification Service | SMTP Error (NewClient with TLS conn): %v", errClient)
			return fmt.Errorf("failed to create SMTP client (SSL/TLS): %w", errClient)
		}
		defer client.Close()

		if errAuth := client.Auth(auth); errAuth != nil {
			log.Printf("Notification Service | SMTP Error (Auth with TLS conn): %v", errAuth)
			return fmt.Errorf("SMTP authentication failed (SSL/TLS): %w", errAuth)
		}
		if errMail := client.Mail(s.senderEmail); errMail != nil {
			log.Printf("Notification Service | SMTP Error (Mail with TLS conn): %v", errMail)
			return fmt.Errorf("SMTP mail command failed (SSL/TLS): %w", errMail)
		}
		for _, recipient := range to {
			if errRcpt := client.Rcpt(recipient); errRcpt != nil {
				log.Printf("Notification Service | SMTP Error (Rcpt %s with TLS conn): %v", recipient, errRcpt)
				return fmt.Errorf("SMTP rcpt command failed for %s (SSL/TLS): %w", recipient, errRcpt)
			}
		}
		w, errData := client.Data()
		if errData != nil {
			log.Printf("Notification Service | SMTP Error (Data with TLS conn): %v", errData)
			return fmt.Errorf("SMTP data command failed (SSL/TLS): %w", errData)
		}
		_, errWrite := w.Write([]byte(msg.String()))
		if errWrite != nil {
			log.Printf("Notification Service | SMTP Error (Write with TLS conn): %v", errWrite)
			return fmt.Errorf("failed to write email body (SSL/TLS): %w", errWrite)
		}
		errCloseData := w.Close()
		if errCloseData != nil {
			log.Printf("Notification Service | SMTP Error (Close Data with TLS conn): %v", errCloseData)
			return fmt.Errorf("failed to close data writer (SSL/TLS): %w", errCloseData)
		}
		err = client.Quit()

	} else { // STARTTLS (e.g., port 587) or plain (e.g., port 25)
		err = smtp.SendMail(addr, auth, s.senderEmail, to, []byte(msg.String()))
	}


	if err != nil {
		log.Printf("Notification Service | SMTP Error sending email to %v: %v", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Notification Service | Email sent successfully to: %v. Subject: %s", to, subject)
	return nil
}