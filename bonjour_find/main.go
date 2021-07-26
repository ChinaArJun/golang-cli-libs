package main

import (
	"log"
	"os"
	"time"

	"github.com/oleksandr/bonjour"
)

func main() {
	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		os.Exit(1)
	}

	results := make(chan *bonjour.ServiceEntry)

	go func(results chan *bonjour.ServiceEntry, exitCh chan<- bool) {
		for e := range results {
			log.Printf("%s", e.Instance)
			exitCh <- true
			time.Sleep(1e9)
			os.Exit(0)
		}
	}(results, resolver.Exit)

	err = resolver.Lookup("DEMO", "_foobar._tcp", "", results)
	if err != nil {
		log.Println("Failed to browse:", err.Error())
	}

	select {}
}
