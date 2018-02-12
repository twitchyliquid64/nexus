package mail

import (
	"context"
	"database/sql"
	"errors"
	"nexus/integration"

	"github.com/twitchyliquid64/smtpd"
)

type integrationServ struct{}

func (s *integrationServ) AcceptRecipient(addr string) error {
	handlerRecipients := integration.EmailTrigger.Recipients()
	if handlerRecipients[addr] {
		return nil
	}
	return errors.New("no handler for address")
}

func (s *integrationServ) Commit(msg []byte, meta *smtpd.MsgMetadata) error {
	return integration.EmailTrigger.HandleMail(msg, meta)
}

// Init starts a SMTP server handling the specified domain.
func Init(ctx context.Context, domain string, db *sql.DB) error {
	s := smtpd.NewServer(":25", domain, make(chan bool), &integrationServ{})
	go s.Start(ctx)
	return nil
}
