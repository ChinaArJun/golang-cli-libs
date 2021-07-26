package main

import (
	"fmt"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleRequest(conn)
	}
}

func handleRequest(inConn net.Conn) {
	fmt.Println("new client")

	outConn, err := net.Dial("tcp", "172.28.0.3:50051")
	if err != nil {
		panic(err)
	}

	fmt.Println("proxy connected")
	chan1 := chanFromConn(inConn)
	chan2 := chanFromConn(outConn)

	for {
		select {
		case b1 := <-chan1:
			if b1 == nil {
				return
			} else {
				outConn.Write(b1)
			}
		case b2 := <-chan2:
			if b2 == nil {
				return
			} else {
				inConn.Write(b2)
			}
		}
	}
}

// chanFromConn creates a channel from a Conn object, and sends everything it
//  Read()s from the socket to the channel.
func chanFromConn(conn net.Conn) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := conn.Read(b)
			fmt.Printf("%v\n\n", b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()

	return c
}
