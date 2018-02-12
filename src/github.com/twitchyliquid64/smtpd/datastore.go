package smtpd

import "container/list"

// DataStore is an interface to get Mailboxes stored in Inbucket
type DataStore interface {
	AcceptRecipient(addr string) error
	Commit(msg []byte, meta *MsgMetadata) error
}

// MsgMetadata encapsulates information about the transfer in which
// an email was received.
type MsgMetadata struct {
	TLS            bool
	From           string
	Domain, Remote string
	Recipients     *list.List
}
