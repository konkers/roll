package main

import "github.com/konkers/roll"

func main() {

	var config roll.Config
	b := roll.NewBot(&config)

	b.Connect()
}
