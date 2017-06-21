package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
)

type config struct {
	Address, Port              string
	Username, Password         string
	BadEmailDomains, BadEmails []string
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

func writeNewConfig(filename string, conf *config) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(conf); err != nil {
		f.Close()
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

	log.Printf("Wrote config back out to %s", filename)
}
