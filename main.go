package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/WesEfird/Monolith/cryptutil"
	"log"
	"net"
	"os"
)

func main() {
	checkRes()
	s := NewServer()
	cons := NewConsole(s)
	// Assign the Console pointer to the Server struct
	s.Cons = cons
	// Load endpoints.json
	s.LoadEndpoints()

	// Start server threads
	go s.ProcessCommands()
	go cons.DisplayConsole()

	cert, err := tls.LoadX509KeyPair("res/server.pem", "res/server.key")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConfig.Rand = rand.Reader
	service := "0.0.0.0:8043"

	// Start the listener service
	listener, err := tls.Listen("tcp", service, &tlsConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
	// Defer closing of listening socket, wrapped in func for error handling. Weird Golang stuff
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("err closing listener socket")
			os.Exit(4)
		}
	}(listener)

	// Accept new connections until program exit
	// Once a connection is accepted, start a new thread to process the connection
	for {
		if conn, err := listener.Accept(); err != nil {
			log.Printf("failed to establish communication with %s", conn.RemoteAddr().String())
			continue
		} else {
			go InitialConnection(s, conn)
		}
	}
}

func checkRes() {
	// Check if directory exists
	if _, err := os.Stat("./res"); os.IsNotExist(err) {
		if err = os.Mkdir("res", os.ModePerm); err != nil {
			fmt.Println("could not create res directory")
		}
	}
	// Check if pem file exists
	if _, err := os.Stat("./res/server.pem"); os.IsNotExist(err) {
		cryptutil.GenerateCerts("server")
		return
	}
	// Check if key file exists
	if _, err := os.Stat("./res/server.key"); os.IsNotExist(err) {
		cryptutil.GenerateCerts("server")
	}
}
