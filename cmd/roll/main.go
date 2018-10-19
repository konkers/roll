package main

import (
	"flag"
	"log"
	"time"

	"github.com/konkers/mocktwitch"
	"github.com/konkers/roll"
	_ "github.com/konkers/roll/modules/alert"
	_ "github.com/konkers/roll/modules/game"
	_ "github.com/konkers/roll/modules/giveaway"
	_ "github.com/konkers/roll/modules/marathon"
	_ "github.com/konkers/roll/modules/simplecmd"
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

	b, err := roll.NewBot(config)
	if err != nil {
		log.Fatalf("Can't create bot: %v", err)
	}
	b.AddModule("alert")
	b.AddModule("game")
	b.AddModule("giveaway")
	b.AddModule("marathon")
	b.AddModule("simplecmd")

	err = b.Connect()
	if err != nil {
		log.Fatalf("Can't connect bot: %v", err)
	}

	// Hack until proper termination control is done.
	for {
		time.Sleep(time.Second)
	}
}
