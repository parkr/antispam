package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func processJunkFolder(c *client.Client, conf *config, mailboxName string, numMessages uint32) {
	// Select mailbox.
	mbox, err := c.Select(mailboxName, false)
	if err != nil {
		panic(err)
	}
	log.Printf("Flags for %s: %+v", mailboxName, mbox.Flags)

	if mbox.Messages == 0 {
		log.Printf("%s is empty", mailboxName)
		return
	}

	// Get the last N messages
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
		for _, sender := range msg.Envelope.From {
			if !spammySender(conf, sender) {
				conf.BadEmailDomains = append(conf.BadEmailDomains, sender.HostName)
			}
			log.Printf("Added %s to list of bad email domains", sender.HostName)
		}
		sort.Strings(conf.BadEmails)
		log.Println(conf.BadEmails)
		spam <- actionRequest{index: i, message: msg, action: "delete"}
		i++
	}
	close(spam)

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

	if err := <-done; err != nil {
		panic(err)
	}
}
