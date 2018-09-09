package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gempir/go-twitch-irc"
	"github.com/konkers/cmd"
)

func main() {

	http.HandleFunc("/", handler)
	log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
	go http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)

	//	cmd := cmd.NewEngine()
	cmd.AddCommand("game", "Tells the channel the current game.", gameCommand, 0)
	cmd.AddCommand("setgame", "Sets the stream game.", setGameCommand, 10)

	bot.IRCClient.OnNewUserstateMessage(func(channel string, user twitch.User, message twitch.Message) {
		if message.Tags["mod"] != "1" {
			log.Printf("WARNING: %s does not have op on %s!", message.Tags["display-name"], channel)
		} else {
			log.Printf("%s has op on %s.", message.Tags["display-name"], channel)
		}

	})

	bot.IRCClient.Join(bot.Config.Channel)

	bot.IRCClient.LogWriter = os.Stdout
	bot.IRCClient.LogPrefix = "[TMI] "
	err := bot.IRCClient.Connect()
	if err != nil {
		panic(err)
	}
}
