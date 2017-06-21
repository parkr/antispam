package main

import (
	"bufio"
	"log"
	"sort"
	"strings"

	_ "github.com/parkr/antispam/statik"
	"github.com/rakyll/statik/fs"
)

var globalDomainBlacklist []string
var globalEmailBlacklist []string

func readGlobalBlacklists() {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
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
