package main

import (
	"fmt"
	"github.com/helmutkemper/pygocentrus"
	"log"
	"os"
)

func main() {

	inPort := os.Getenv("IN_PORT")
	outAddress := os.Getenv("OUT_ADDR")
	outPort := os.Getenv("OUT_PORT")

	fmt.Printf("pygocentrus listen(%v)\n", ":"+inPort)
	fmt.Printf("pygocentrus dial(%v)\n", ":"+outAddress+":"+outPort)

	listen := pygocentrus.Listen{
		In: pygocentrus.Connection{
			Address:  ":" + inPort,
			Protocol: "tcp",
		},
		Out: pygocentrus.Connection{
			Address:  outAddress + ":" + outPort,
			Protocol: "tcp",
		},
		Pygocentrus: pygocentrus.Pygocentrus{
			Enabled: true,
			ChangeContent: pygocentrus.ChangeContent{
				Rate:           0.1,
				ChangeBytesMin: 1,
				ChangeBytesMax: 10,
			},
		},
	}
	err := listen.Listen()
	if err != nil {
		log.Panic(err.Error())
	}

}
