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

var globalDomainBlacklist []string
var globalEmailBlacklist []string

func readGlobalBlacklists() {
	statikFS, err := fs.New()
	if err != nil {
		panic(errors.Wrap(err, "unable to register statik fs"))
	}

	domainsBaseChan := make(chan string, 10000)
	domainsChan := make(chan string, 500)
	emailsChan := make(chan string, 500)

	go readBlacklistFile(statikFS, domainsBaseChan, "/dom-bl-base.txt")
	go readBlacklistFile(statikFS, domainsChan, "/dom-bl.txt")
	go readBlacklistFile(statikFS, emailsChan, "/from-bl.txt")

	for domain := range domainsBaseChan {
		globalDomainBlacklist = append(globalDomainBlacklist, domain)
	}

	for domain := range domainsChan {
		globalDomainBlacklist = append(globalDomainBlacklist, domain)
	}

	for email := range emailsChan {
		globalEmailBlacklist = append(globalEmailBlacklist, email)
	}

	sort.Strings(globalDomainBlacklist)
	sort.Strings(globalEmailBlacklist)
}

func readBlacklistFile(statikFS http.FileSystem, contentsChan chan string, filename string) {
	// Read the base domain blacklist. This changes infrequently.
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
