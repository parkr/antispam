package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
)

type config struct {
	Address, Port              string   `json:",omitempty"`
	Username, Password         string   `json:",omitempty"`
	BadEmailDomains, BadEmails []string `json:",omitempty"`
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

func readFilterFile(conf *config, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(conf); err != nil && err != io.EOF {
		return err
	}

	sort.Strings(conf.BadEmailDomains)
	sort.Strings(conf.BadEmails)

	return nil
}

func writeNewFilterFile(filename string, conf *config) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
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
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

	log.Printf("Wrote filter config back out to %s", filename)
}
