package main

import (
	"github.com/grandcat/zeroconf"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	server, err := zeroconf.Register("GoZeroconf", "_workstation._tcp", "local.", 42424, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}
	defer server.Shutdown()

	time.Sleep(5 * time.Second)

	server, err = zeroconf.Register("GoZeroconf", "_workstation_2._tcp", "local.", 42424, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {}
}
