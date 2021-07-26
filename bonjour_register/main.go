package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/oleksandr/bonjour"
)

func main() {
	// Run registration (blocking call)
	s, err := bonjour.Register("Foo Service", "_foobar._tcp", "", 9999, []string{"txtv=1", "app=test"}, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// Ctrl+C handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt)
	for sig := range handler {
		if sig == os.Interrupt {
			s.Shutdown()
			time.Sleep(1e9)
			break
		}
	}
}
