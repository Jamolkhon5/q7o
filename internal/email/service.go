package email

import (
	"fmt"
	"gopkg.in/gomail.v2"
	"q7o/config"
)

type Service struct {
	cfg config.SMTPConfig
}

func NewService(cfg config.SMTPConfig) *Service {
	return &Service{
		cfg: cfg,
	}
}

func (s *Service) SendVerificationEmail(to, username, code string) error {
	subject := "Verify your Q7O account"
	body := fmt.Sprintf(`
        <h2>Welcome to Q7O, %s!</h2>
        <p>Your verification code is:</p>
        <h1 style="color: #4CAF50; letter-spacing: 5px;">%s</h1>
        <p>This code will expire in 15 minutes.</p>
        <p>If you didn't create an account, please ignore this email.</p>
    `, username, code)

	return s.sendEmail(to, subject, body)
}

func (s *Service) SendCallMissedEmail(to, callerName string) error {
	subject := "Missed call on Q7O"
	body := fmt.Sprintf(`
        <h2>You have a missed call</h2>
        <p>%s tried to call you on Q7O.</p>
        <p>Log in to call them back!</p>
    `, callerName)

	return s.sendEmail(to, subject, body)
}

func (s *Service) sendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Pass)

	return d.DialAndSend(m)
}
