package main

import (
	"flag"
	"log"

	"github.com/konkers/mocktwitch"
	"github.com/konkers/roll"
)

var configFileName = flag.String("config", "config.json", "Config file")
var testServer = flag.Bool("test", false, "Enables mocked twitch server")

func main() {

	config, err := roll.LoadConfig(*configFileName)
	if err != nil {
		log.Fatalf("Can't load Config: %v", err)
	}

	var mock *mocktwitch.Twitch
	if *testServer {
		mock, err = mocktwitch.NewTwitch()
		if err != nil {
			log.Fatalf("Can't create mock twitch: %v.", err)
		}
		config.IRCAddress = mock.IrcHost
		config.APIURLBase = mock.ApiUrlBase
	}

	b := roll.NewBot(config)
	b.Connect()
}
