package main

import (
	"log"
)

func main() {
	app, err := NewDaemon()
	if err != nil {
		log.Fatal(err)
	}

	// Start daemon processes
	go app.RunDaemon()
	app.api.StartAPI()
}
