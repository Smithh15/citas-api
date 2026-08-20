package mailer

import "log"

type Mailer interface {
	Send(to, subject, body string) error
}

type LogMailer struct{}

func (m *LogMailer) Send(to, subject, body string) error {
	log.Printf("[EMAIL] to=%s subject=%q body=%s", to, subject, body)
	return nil
}
