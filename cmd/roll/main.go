package main

import (
	"flag"
	"log"

	"github.com/konkers/roll"
)

var configFileName = flag.String("config", "config.json", "Config file")

func main() {

	config, err := roll.LoadConfig(*configFileName)
	if err != nil {
		log.Fatalf("Can't load Config: %v", err)
	}
	b := roll.NewBot(config)

	b.Connect()
}
