package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jhillyerd/enmime"
	"github.com/twitchyliquid64/smtpd"
)

type sorter struct{}

func (s *sorter) AcceptRecipient(addr string) error {
	return nil
}

func (s *sorter) Commit(msg []byte, meta *smtpd.MsgMetadata) error {
	env, err := enmime.ReadEnvelope(bytes.NewBuffer(msg))
	if err != nil {
		fmt.Printf("Envelope error: %v\n", err)
		return err
	}
	fmt.Printf("Meta: %+v\n", meta)
	fmt.Printf("Subject: %s\n", env.GetHeader("Subject"))
	fmt.Printf("Content: %s\n\n", env.Text)
	return nil
}

type printer struct{}

func (p *printer) AcceptRecipient(addr string) error {
	return nil
}

func (p *printer) Commit(msg []byte, meta *smtpd.MsgMetadata) error {
	fmt.Println("Got message!")
	fmt.Printf("Meta: %+v\n", meta)
	fmt.Printf("Message: \n%s\n\n", string(msg))
	return nil
}

func main() {
	dieChan := make(chan bool)
	ctx := context.Background()
	s := smtpd.NewServer(":25", os.Args[1], dieChan, &sorter{})
	s.Start(ctx)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	s.Drain()
}
