smtpd
=======

Barebones inbound email server.

## Usage

1. Call `NewServer()` with listener + server parameters.
2. Implement the interface `DataStore`, which will be invoked when saving messages.

`DataStore.Commit([]byte, *MsgMetadata)` will be called when saving a message.

`DataStore.AcceptRecipient(string)` will be called during recipient exchange, and will decline
that recipient if a non-nil error is returned.

## License

This has been bastardized from https://github.com/jhillyerd/inbucket/tree/master/smtpd

Inbucket project is MIT, therefore this one is too.
