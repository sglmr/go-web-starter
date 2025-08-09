package email

import (
	"bytes"
	"fmt"
	"log/slog"
	"time"

	"github.com/sglmr/gowebstart/assets"
	"github.com/sglmr/gowebstart/internal/funcs"
	"github.com/wneessen/go-mail"

	htmlTemplate "html/template"
	textTemplate "text/template"
)

const defaultTimeout = 10 * time.Second

type Attachment struct {
	Filename string
	Data     []byte
}

// MailerInterface enables exchanging between a Mailer and LogMailer.
type MailerInterface interface {
	Send(recipient string, replyTo string, data any, templates ...string) error
	SendWithAttachment(recipient, replyTo string, data any, attachment Attachment, templates ...string) error
}

//	Email Mailer

// Mailer that sends SMTP emails
type Mailer struct {
	client *mail.Client
	from   string
}

// NewMailer creates and returns a new Mailer for sending emails via SMTP. It
// configures the SMTP client with the provided host, port, username, password,
// and sender address. This function is the entry point for setting up the
// email sending service.
func NewMailer(host string, port int, username, password, from string) (*Mailer, error) {
	client, err := mail.NewClient(host, mail.WithTimeout(defaultTimeout), mail.WithSMTPAuth(mail.SMTPAuthLogin), mail.WithPort(port), mail.WithUsername(username), mail.WithPassword(password))
	if err != nil {
		return nil, err
	}

	mailer := &Mailer{
		client: client,
		from:   from,
	}

	return mailer, nil
}

// Send sends an email to a specified recipient. It uses templates to generate
// the email's subject, plain-text body, and HTML body. The method handles all
// aspects of email composition and sending, including retries in case of
// failure.
func (m *Mailer) Send(recipient string, replyTo string, data any, templates ...string) error {
	// Create a slice from the patterns argument
	for i := range templates {
		// templates[i] = "emails/" + templates[i]
		templates[i] = "emails/" + templates[i]
	}

	// Initialize a new mail message
	msg := mail.NewMsg()

	err := msg.To(recipient)
	if err != nil {
		return err
	}

	if len(replyTo) > 0 {
		err = msg.ReplyTo(replyTo)
		if err != nil {
			return err
		}
	}

	err = msg.From(m.from)
	if err != nil {
		return err
	}

	ts, err := textTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, templates...)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = ts.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	msg.Subject(subject.String())

	plainBody := new(bytes.Buffer)
	err = ts.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())

	if ts.Lookup("htmlBody") != nil {
		ts, err := htmlTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, templates...)
		if err != nil {
			return err
		}

		htmlBody := new(bytes.Buffer)
		err = ts.ExecuteTemplate(htmlBody, "htmlBody", data)
		if err != nil {
			return err
		}

		msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())
	}

	// Retry up to 3 times
	for i := 1; i <= 3; i++ {
		err = m.client.DialAndSend(msg)

		if nil == err {
			return nil
		}

		if i != 3 {
			time.Sleep(2 * time.Second)
		}
	}

	return err
}

// SendWithAttachment sends an email with an attachment. It extends the Send
// method by allowing a file to be attached to the email. This is useful for
// sending reports, invoices, or other documents.
func (m *Mailer) SendWithAttachment(
	recipient, replyTo string,
	data any,
	attachment Attachment,
	templates ...string,
) error {
	// Create a slice from the patterns argument
	for i := range templates {
		templates[i] = "emails/" + templates[i]
	}

	// Initialize a new mail message
	msg := mail.NewMsg()

	err := msg.To(recipient)
	if err != nil {
		return err
	}

	if len(replyTo) > 0 {
		err = msg.ReplyTo(replyTo)
		if err != nil {
			return err
		}
	}

	err = msg.From(m.from)
	if err != nil {
		return err
	}

	ts, err := textTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, templates...)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	if err := ts.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}
	msg.Subject(subject.String())

	plainBody := new(bytes.Buffer)
	if err := ts.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())

	if ts.Lookup("htmlBody") != nil {
		ts, err := htmlTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, templates...)
		if err != nil {
			return err
		}

		htmlBody := new(bytes.Buffer)
		if err := ts.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
			return err
		}

		msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())
	}

	// Add the CSV as an attachment
	err = msg.AttachReader(attachment.Filename, bytes.NewReader(attachment.Data))
	if err != nil {
		return fmt.Errorf("failed to attach CSV: %w", err)
	}

	// Retry up to 3 times
	for i := 1; i <= 3; i++ {
		err = m.client.DialAndSend(msg)

		if nil == err {
			return nil
		}

		if i != 3 {
			time.Sleep(2 * time.Second)
		}
	}

	return err
}

//	Log Mailer

// LogMailer object for logging emails instead of sending them
type LogMailer struct {
	log *slog.Logger
}

// NewLogMailer creates a new LogMailer that logs emails instead of sending them.
// This is useful for development and testing environments where you don't want
// to send real emails. It takes a logger as input and returns a LogMailer.
func NewLogMailer(l *slog.Logger) *LogMailer {
	return &LogMailer{
		log: l,
	}
}

// Send logs the email details to the logger instead of sending it. This method
// is part of the MailerInterface and is used by the LogMailer to simulate
// sending an email. It logs the recipient, reply-to address, templates, and
// data.
func (m *LogMailer) Send(recipient string, replyTo string, data any, templates ...string) error {
	m.log.Info("send email", "recipient", recipient, "replyTo", replyTo, "templates", templates, "data", data)
	return nil
}

// SendWithAttachment logs the email details and attachment information to the
// logger. This method is part of the MailerInterface and is used by the
// LogMailer to simulate sending an email with an attachment. It logs the
// recipient, reply-to address, templates, attachment filename, and data.
func (m *LogMailer) SendWithAttachment(
	recipient, replyTo string,
	data any,
	attachment Attachment,
	templates ...string,
) error {
	m.log.Info("send email with attachment", "recipient", recipient, "replyTo", replyTo, "templates", templates, "attachment", attachment.Filename, "data", data)

	return nil
}
