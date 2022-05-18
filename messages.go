package main

import (
	"fmt"
)

func (e *Endpoint) msgHost(s *Server, msgArgs []string) {
	if len(msgArgs) < 2 {
		fmt.Println("malformed HOST message")
		return
	}
	// Cache Hostname in this Endpoint struct
	e.Hostname = msgArgs[1]
	// Send command to the server to save the updated Endpoint information
	s.Commands <- "SAVE"
}

func (e *Endpoint) msgPong() {
	//fmt.Printf("Received ping from %s\n", e.Conn.RemoteAddr().String())
	return
}
