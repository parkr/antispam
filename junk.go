package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// shouldBanEntireDomain determines whether the sender's HostName should be
// blocked, or whether it should be scoped down to just the mailbox on that
// hostname.
// Returns true if an entire domain should be blocked, or false if just the
// mailbox should be blocked.
func shouldBanEntireDomain(sender *imap.Address) bool {
	switch sender.HostName {
	case "gmail.com", "yahoo.com", "outlook.com", "hotmail.com":
		return false
	}
	return true
}

// banSender adds the sender to the custom block list.
func banSender(conf *config, sender *imap.Address) {
	if shouldBanEntireDomain(sender) {
		conf.BadEmailDomains = append(conf.BadEmailDomains, sender.HostName)
	} else {
		conf.BadEmails = append(conf.BadEmails, sender.MailboxName+"@"+sender.HostName)
	}
}

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
	log.Printf("Loading messages from %d to %d", from, to)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, numMessages)
	spam := make(chan actionRequest, numMessages)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	log.Printf("Last %d messages:", numMessages)
	i := from
	for msg := range messages {
		log.Println("*", msg.Envelope.Subject)
		for _, sender := range msg.Envelope.From {
			if !spammySender(conf, sender) {
				banSender(conf, sender)
				log.Printf("Added %s to list of bad email domains", sender.HostName)
			}
		}
		for _, sender := range msg.Envelope.Sender {
			if !spammySender(conf, sender) {
				banSender(conf, sender)
				log.Printf("Added %s to list of bad email domains", sender.HostName)
			}
		}
		sort.Strings(conf.BadEmails)
		sort.Strings(conf.BadEmailDomains)
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
