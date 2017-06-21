package main

import (
	"fmt"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
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
	return false
}

func spammySender(conf *config, sender *imap.Address) bool {
	senderEmail := sender.MailboxName + "@" + sender.HostName
	senderEmailAtDomain := sender.MailboxName + "@" + sender.AtDomainList

	if isInStringSlice(conf.BadEmailDomains, sender.HostName) ||
		isInStringSlice(conf.BadEmailDomains, sender.AtDomainList) ||
		isInStringSlice(globalDomainBlacklist, sender.HostName) ||
		isInStringSlice(globalDomainBlacklist, sender.AtDomainList) ||
		isInStringSlice(conf.BadEmails, senderEmail) ||
		isInStringSlice(conf.BadEmails, senderEmailAtDomain) ||
		isInStringSlice(globalEmailBlacklist, senderEmail) ||
		isInStringSlice(globalEmailBlacklist, senderEmailAtDomain) {
		return true
	}

	return false
}

func processInbox(c *client.Client, conf *config, numMessages uint32) {
	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		panic(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	if mbox.Messages == 0 {
		log.Println("INBOX is empty")
		return
	}

	// Get the last 10 messages
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
		done <- c.Fetch(seqset, []string{imap.EnvelopeMsgAttr}, messages)
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
		panic(err)
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
			panic(fmt.Errorf("What does '%s' mean?", spammy.action))
		}
	}
}
