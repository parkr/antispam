package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"

	"github.com/pkg/errors"
)

// DefaultConfig is the path for the default config file.
const DefaultConfig = "config.json"

type config struct {
	Address, Port              string   `json:",omitempty"`
	Username, Password         string   `json:",omitempty"`
	BadEmailDomains, BadEmails []string `json:",omitempty"`
	UseJunk                    bool     `json:",omitempty"`
	UseSpam                    bool     `json:",omitempty"`
	UseFlags                   bool     `json:",omitempty"`
	UseBlockLists              bool     `json:",omitempty"`
}

func readConfigFile(conf *config, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "unable to open file %q", filename)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(conf); err != nil && err != io.EOF {
		return errors.Wrapf(err, "unable to read json from %q", filename)
	}

	sort.Strings(conf.BadEmailDomains)
	sort.Strings(conf.BadEmails)

	return nil
}

func writeNewFilterFile(filename string, conf *config) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(errors.Wrapf(err, "unable to open file for writing %q", filename))
	}

	sort.Strings(conf.BadEmailDomains)
	sort.Strings(conf.BadEmails)

	filterConf := &config{
		BadEmailDomains: conf.BadEmailDomains,
		BadEmails:       conf.BadEmails,
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(filterConf); err != nil {
		f.Close()
		panic(errors.Wrapf(err, "unable to write json to %q", filename))
	}

	if err := f.Close(); err != nil {
		panic(errors.Wrapf(err, "unable to close file descriptor for %q", filename))
	}

	log.Printf("Wrote filter config back out to %q", filename)
}
