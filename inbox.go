package main

import (
	"fmt"
	"log"

	imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
)

func spammyMessage(conf *config, msg *imap.Message) bool {
	for _, fromHeader := range msg.Envelope.From {
		if spammySender(conf, fromHeader) {
			return true
		}
		log.Println("  ", fromHeader.MailboxName+"@"+fromHeader.HostName, "not spammy")
	}
	for _, senderHeader := range msg.Envelope.Sender {
		if spammySender(conf, senderHeader) {
			return true
		}
		log.Println("  ", senderHeader.MailboxName+"@"+senderHeader.HostName, "not spammy")
	}

	if conf.UseFlags && spammyFlags(conf, msg.Flags) {
		return true
	}
	return false
}

func spammySender(conf *config, sender *imap.Address) bool {
	senderEmail := sender.MailboxName + "@" + sender.HostName
	senderEmailAtDomain := sender.MailboxName + "@" + sender.AtDomainList

	if isInStringSlice(conf.BadEmailDomains, sender.HostName) ||
		isInStringSlice(conf.BadEmailDomains, sender.AtDomainList) ||
		isInStringSlice(globalDomainBlocklist, sender.HostName) ||
		isInStringSlice(globalDomainBlocklist, sender.AtDomainList) ||
		isInStringSlice(conf.BadEmails, senderEmail) ||
		isInStringSlice(conf.BadEmails, senderEmailAtDomain) ||
		isInStringSlice(globalEmailBlocklist, senderEmail) ||
		isInStringSlice(globalEmailBlocklist, senderEmailAtDomain) {
		return true
	}

	return false
}

func spammyFlags(conf *config, flags []string) bool {
	flagged := false
	seen := false

	for _, flag := range flags {
		switch flag {
		case imap.FlaggedFlag:
			flagged = true
		case imap.SeenFlag:
			seen = true
		}
	}

	// it's considered spam if flagged but not read
	if flagged && !seen {
		return true
	}

	return false
}

func processInbox(c *client.Client, conf *config, numMessages uint32) {
	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		panic(errors.Wrap(err, "error selecting INBOX"))
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	if mbox.Messages == 0 {
		log.Println("INBOX is empty")
		return
	}

	// Get the last numMessages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > numMessages {
		// We're using unsigned integers here, only substract if the result is > 0
		from = mbox.Messages - numMessages
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, numMessages)
	spam := make(chan actionRequest, numMessages)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags}, messages)
	}()

	log.Printf("Last %d messages:", numMessages)
	i := from
	for msg := range messages {
		log.Println("*", msg.Envelope.Subject)
		if spammyMessage(conf, msg) {
			spam <- actionRequest{index: i, message: msg, action: "delete"}
		}
		i++
	}
	close(spam)

	if err := <-done; err != nil {
		panic(errors.Wrap(err, "error fetching messages in inbox"))
	}

	numDeleted := uint32(0)
	for spammy := range spam {
		log.Printf("* SPAM: %s (index=%d) (action=%s)", spammy.message.Envelope.Subject, spammy.index, spammy.action)
		switch spammy.action {
		case "delete":
			deleteMessage(c, spammy.index-numDeleted)
			log.Println("Deleted", spammy.message.Envelope.Subject)
			numDeleted++
		default:
			panic(fmt.Errorf("unhandled spammy action %q", spammy.action))
		}
	}
}
