package main

import (
	"bufio"
	"log"
	"net/http"
	"sort"
	"strings"

	_ "github.com/parkr/antispam/statik"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"
)

var globalDomainBlocklist []string
var globalEmailBlocklist []string

func readGlobalBlocklists() {
	statikFS, err := fs.New()
	if err != nil {
		panic(errors.Wrap(err, "unable to register statik fs"))
	}

	domainsBaseChan := make(chan string, 10000)
	domainsChan := make(chan string, 500)
	emailsChan := make(chan string, 500)

	go readBlocklistFile(statikFS, domainsBaseChan, "/dom-bl-base.txt")
	go readBlocklistFile(statikFS, domainsChan, "/dom-bl.txt")
	go readBlocklistFile(statikFS, emailsChan, "/from-bl.txt")

	for domain := range domainsBaseChan {
		globalDomainBlocklist = append(globalDomainBlocklist, domain)
	}

	for domain := range domainsChan {
		globalDomainBlocklist = append(globalDomainBlocklist, domain)
	}

	for email := range emailsChan {
		globalEmailBlocklist = append(globalEmailBlocklist, email)
	}

	sort.Strings(globalDomainBlocklist)
	sort.Strings(globalEmailBlocklist)
}

func readBlocklistFile(statikFS http.FileSystem, contentsChan chan string, filename string) {
	// Read the base domain blocklist. This changes infrequently.
	f, err := statikFS.Open(filename)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			pieces := strings.Split(scanner.Text(), ";")
			if len(pieces) > 0 {
				contentsChan <- pieces[0]
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("%+v", errors.Wrapf(err, "error scanning %q", filename))
		}
	} else {
		log.Printf("%+v", errors.Wrapf(err, "unable to open %q", filename))
	}
	close(contentsChan)
}
