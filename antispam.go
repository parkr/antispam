package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	_ "github.com/parkr/antispam/statik"
	"github.com/rakyll/statik/fs"
)

var globalDomainBlacklist []string
var globalEmailBlacklist []string

type config struct {
	Address, Port              string
	Username, Password         string
	BadEmailDomains, BadEmails []string
}

type actionRequest struct {
	index   uint32
	message *imap.Message
	action  string
}

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

func isInStringSlice(haystack []string, needle string) bool {
	i := sort.SearchStrings(haystack, needle)
	return i < len(haystack) && haystack[i] == needle
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

func readConfig(filename string) *config {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	conf := &config{}
	if err = json.NewDecoder(f).Decode(conf); err != nil {
		panic(err)
	}

	sort.Strings(conf.BadEmailDomains)
	sort.Strings(conf.BadEmails)

	return conf
}

func readGlobalBlacklists() {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	// Read the base domain blacklist. This changes infrequently.
	f, err := statikFS.Open("/dom-bl-base.txt")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			pieces := strings.Split(scanner.Text(), ";")
			if len(pieces) > 0 {
				globalDomainBlacklist = append(globalDomainBlacklist, pieces[0])
			}
		}
	} else {
		log.Println("error reading dom-bl-base.txt:", err)
	}

	// Read the more up-to-date domain blacklist subset.
	f, err = statikFS.Open("/dom-bl.txt")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			pieces := strings.Split(scanner.Text(), ";")
			if len(pieces) > 0 {
				globalDomainBlacklist = append(globalDomainBlacklist, pieces[0])
			}
		}
	} else {
		log.Println("error reading dom-bl.txt:", err)
	}

	// Read the email blacklist.
	f, err = statikFS.Open("/from-bl.txt")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			pieces := strings.Split(scanner.Text(), ";")
			if len(pieces) > 0 {
				globalEmailBlacklist = append(globalEmailBlacklist, pieces[0])
			}
		}
	} else {
		log.Println("error reading from-bl.txt:", err)
	}

	sort.Strings(globalDomainBlacklist)
	sort.Strings(globalEmailBlacklist)
}

func deleteMessage(c *client.Client, messageIndex uint32) {
	seqset := new(imap.SeqSet)
	seqset.AddRange(messageIndex, messageIndex)

	// First mark the message as deleted
	operation := "+FLAGS.SILENT"
	flags := []interface{}{imap.DeletedFlag}
	if err := c.Store(seqset, operation, flags, nil); err != nil {
		log.Fatal(err)
	}

	// Then delete it
	if err := c.Expunge(nil); err != nil {
		log.Fatal(err)
	}
}

func main() {
	confFile := flag.String("config", "", "Path to config file")
	numMessagesFromFlag := flag.Uint("num", 10, "Number of messages to process per execution")
	flag.Parse()

	if *confFile == "" {
		panic("Specify the -config flag")
	}

	log.Println("Reading config...")
	conf := readConfig(*confFile)

	log.Println("Loading global blacklists...")
	readGlobalBlacklists()

	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS(conf.Address+":"+conf.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(conf.Username, conf.Password); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 10 messages
	from := uint32(1)
	to := mbox.Messages
	numMessages := uint32(*numMessagesFromFlag)
	if mbox.Messages > numMessages {
		// We're using unsigned integers here, only substract if the result is > 0
		from = mbox.Messages - numMessages
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, numMessages)
	spam := make(chan actionRequest, numMessages)
	done = make(chan error, 1)
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

	numDeleted := uint32(0)
	for spammy := range spam {
		log.Printf("* SPAM: %s (index=%d) (action=%s)", spammy.message.Envelope.Subject, spammy.index, spammy.action)
		switch spammy.action {
		case "delete":
			deleteMessage(c, spammy.index-numDeleted)
			log.Println("Deleted", spammy.message.Envelope.Subject)
			numDeleted++
		default:
			log.Fatalf("What does '%s' mean?", spammy.action)
		}
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")
}
