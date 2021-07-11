package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
)

var defaultFilterFile = "/tmp/antispam-filter.json"

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
	operation := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	if err := c.Store(seqset, operation, flags, nil); err != nil {
		panic(errors.Wrapf(err, "error marking message at index %d as deleted", messageIndex))
	}

	// Then clear FlaggedFlag
	operation = imap.FormatFlagsOp(imap.RemoveFlags, true)
	flags = []interface{}{imap.FlaggedFlag}

	if err := c.Store(seqset, operation, flags, nil); err != nil {
		panic(errors.Wrapf(err, "error clearing flagged flag at index %d", messageIndex))
	}

	// Then delete it
	if err := c.Expunge(nil); err != nil {
		panic(errors.Wrap(err, "error expunging deleted messages"))
	}
}

func printOutput(output io.Reader) {
	if output == nil || output == os.Stderr {
		return
	}

	outputBytes, err := ioutil.ReadAll(output)
	if err != nil {
		fmt.Printf("error reading output buffer: %+v\n", errors.Wrap(err, "error reeading all output"))
		return
	}

	fmt.Print(string(outputBytes))
}

func main() {
	var output io.ReadWriter

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				fmt.Printf("panic: %+v\n", errors.Wrap(err, "fatal error occurred"))
			} else {
				fmt.Printf("panic: %+v\n", r)
			}
			printOutput(output)
		}
	}()

	debugFlag := flag.Bool("debug", false, "Whether to print debug info")
	confFile := flag.String("config", "", "Path to config file")
	filterFile := flag.String("filter", defaultFilterFile, "Path to filter file (must be read-writable)")
	numMessagesFromFlag := flag.Uint("num", 10, "Number of messages to process per execution")
	flag.Parse()

	if *confFile == "" {
		// If DefaultConfig works, use it, else panic
		f, err := os.Open(DefaultConfig)
		if err != nil {
			panic("Specify the -config flag")
		}
		f.Close()

		*confFile = DefaultConfig
	}

	if *debugFlag {
		output = os.Stderr
	} else {
		output = &bytes.Buffer{}
	}
	log.SetOutput(output)

	if *filterFile == "" {
		filterFile = &defaultFilterFile
	}

	conf := &config{UseSpam: true, UseJunk: true, UseBlockLists: true}
	log.Println("Reading config...")
	if err := readConfigFile(conf, *confFile); err != nil {
		panic(err)
	}
	log.Println("Read config", conf)

	log.Printf("Reading filter file %s", *filterFile)
	if err := readConfigFile(conf, *filterFile); err != nil {
		panic(err)
	}
	log.Println("Read filter", conf)

	if conf.UseBlockLists {
		log.Println("Loading global blocklists...")
		readGlobalBlocklists()
	} else {
		log.Println("Skipping global blocklists...")
	}

	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS(conf.Address+":"+conf.Port, nil)
	if err != nil {
		panic(errors.Wrapf(err, "unable to dial tls %q", conf.Address+":"+conf.Port))
	}
	log.Println("Connected")
	c.ErrorLog = log.New(output, "imap/client: ", log.LstdFlags)
	c.SetDebug(output)

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(conf.Username, conf.Password); err != nil {
		panic(errors.Wrapf(err, "unable to login as %q", conf.Username))
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
		panic(errors.Wrap(err, "unable to list mailboxes"))
	}
	close(done)

	numMessages := uint32(*numMessagesFromFlag)

	if conf.UseJunk {
		processJunkFolder(c, conf, "Junk", numMessages) // Things we manually label as Junk will be added to our config.
	}
	if conf.UseSpam {
		processJunkFolder(c, conf, "Spam", numMessages) // Things we manually label as Junk will be added to our config.
	}
	processInbox(c, conf, numMessages) // Remove spam from the inbox.

	if conf.UseBlockLists {
		writeNewFilterFile(*filterFile, conf)
	}

	log.Println("Done!")
}
