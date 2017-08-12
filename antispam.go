package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type actionRequest struct {
	index   uint32
	message *imap.Message
	action  string
}

func isInStringSlice(haystack []string, needle string) bool {
	i := sort.SearchStrings(haystack, needle)
	return i < len(haystack) && haystack[i] == needle
}

func deleteMessage(c *client.Client, messageIndex uint32) {
	seqset := new(imap.SeqSet)
	seqset.AddRange(messageIndex, messageIndex)

	// First mark the message as deleted
	operation := "+FLAGS.SILENT"
	flags := []interface{}{imap.DeletedFlag}
	if err := c.Store(seqset, operation, flags, nil); err != nil {
		panic(err)
	}

	// Then delete it
	if err := c.Expunge(nil); err != nil {
		panic(err)
	}
}

func main() {
	var output bytes.Buffer
	log.SetOutput(&output)
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("An error occured! %+v\n", r)
			fmt.Print(output.String())
		}
	}()

	debugFlag := flag.Bool("debug", false, "Whether to print debug info")
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
		panic(err)
	}
	log.Println("Connected")
	c.ErrorLog = log.New(&output, "imap/client: ", log.LstdFlags)
	c.SetDebug(&output)

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(conf.Username, conf.Password); err != nil {
		panic(err)
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
		panic(err)
	}
	close(done)

	numMessages := uint32(*numMessagesFromFlag)
	processJunkFolder(c, conf, "Junk", numMessages) // Things we manually label as Junk will be added to our config.
	processJunkFolder(c, conf, "Spam", numMessages) // Things we manually label as Junk will be added to our config.
	processInbox(c, conf, numMessages)              // Remove spam from the inbox.

	writeNewConfig(*confFile, conf)

	log.Println("Done!")

	if *debugFlag == true {
		fmt.Print(output.String())
	}
}
