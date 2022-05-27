package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Endpoint contains information relevant to tracking and communicating with remote endpoints
type Endpoint struct {
	Conn        net.Conn
	Hostname    string
	Commands    chan string
	EndpointID  string
	MsgData     []string
	AuthString  string
	FileQueue   map[string]*FileData
	UploadQueue []string
}

// SaveData contains information relevant to saving Endpoint data to a file or database
type SaveData struct {
	EndpointID string
	Hostname   string
	AuthString string
}

type FileData struct {
	LocalPath string
	ShaHash   string
}

// NewEndpoint will return a pointer to an Endpoint struct. If an Endpoint with its corresponding EndpointId already
// exists in the servers Endpoints slice, then this pointer will be returned and the cached connection will be updated.
// Otherwise, a new Endpoint struct will be created and added to the servers Endpoint slice, and this pointer will be
// returned.
func NewEndpoint(s *Server, connection net.Conn, endpointId string) *Endpoint {
	if e, err := s.GetEndpointById(endpointId); err != nil {
		newEndpoint := &Endpoint{
			Conn:        connection,
			EndpointID:  endpointId,
			Commands:    make(chan string, 50),
			MsgData:     []string{},
			FileQueue:   make(map[string]*FileData),
			UploadQueue: []string{},
		}
		s.AddEndpoint(newEndpoint)
		return newEndpoint
	} else {
		e.Conn = connection
		return e
	}
}

func InitialConnection(s *Server, conn net.Conn) {
	defer conn.Close()
	data, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		conn.Close()
		fmt.Println("err receiving initial connection data")
		return
	}

	data = strings.TrimSpace(data)
	msgType := strings.Split(data, "\x02")[0]
	var msgArgs []string
	if len(strings.Split(data, "\x02")) < 2 {
		msgArgs = []string{}
	} else {
		msgArgs = strings.Split(data, "\x02")[1:]
	}
	if err := verifyBeaconMessage(msgType, msgArgs); err != nil {
		conn.Close()
		return
	}
	endpointID := msgArgs[1]
	// Verify the AuthString matches the one that is stored for this endpoint
	if res := s.VerifyAuthString(endpointID, msgArgs[2]); res != true {
		conn.Close()
		return
	}
	e := NewEndpoint(s, conn, endpointID)
	// Check if there is a hostname associated with this endpoint, if not, then buffer a HOST command to be sent
	if e.Hostname == "" {
		e.BufferCommand("HOST")
	}
	// Send all buffered commands
	e.SendCommands()
	// Read response from endpoint
	e.ReadMessages(s)
}

func (e *Endpoint) ReadMessages(s *Server) {
	for {
		data, err := bufio.NewReader(e.Conn).ReadString('\n')
		if err != nil {
			return
		}
		// Trim '\n' byte
		data = strings.TrimSpace(data)
		// Get the msgType from the first portion of data in the packet
		msgType := strings.Split(data, "\x02")[0]
		var msgArgs []string
		// Get any message arguments from the rest of the packet, if they exist
		if len(strings.Split(data, "\x02")) < 2 {
			msgArgs = []string{}
		} else {
			msgArgs = strings.Split(data, "\x02")[1:]
		}

		e.processMessage(s, msgType, msgArgs)
	}
}

func (e *Endpoint) SendCommands() {
	// Get how many commands are in the Commands channel buffer
	cmdsLen := len(e.Commands)

	// If there are no commands to send, then let the endpoint know by sending a BUF packet with a value of 0
	if cmdsLen == 0 {
		e.Conn.Write([]byte("CMD:\x02BUF\x020\n"))
		return
	}

	// Push a done signal to the Commands buffer to ensure the thread knows when to stop reading from the Commands buffer
	// Otherwise the thread will continue to listen to the Commands channel indefinitely
	e.Commands <- "CMDEND\n"
	// Send a BUF packet to the endpoint so the endpoint will know how many commands to expect
	e.Conn.Write([]byte("CMD:\x02BUF\x02" + strconv.Itoa(cmdsLen) + "\n"))
	for cmd := range e.Commands {
		if cmd == "CMDEND\n" {
			return
		}
		e.Conn.Write([]byte(cmd))
	}
}

// BufferCommand will take in a command and arguments then will build the proper command string to be sent to
// the endpoint. The built command string will be buffered in the Commands channel.
func (e *Endpoint) BufferCommand(cmd string, args ...string) {
	builtCmd := cmd
	// If args were provided, then join the args by the \x02 byte and append them to the command
	if len(args) > 0 {
		for _, s := range args {
			builtCmd += "\x02" + s
		}
	}
	// Prepend the "CMD:" identifier and append the termination "\n" byte
	builtCmd = strings.Join([]string{"CMD:\x02", builtCmd, "\n"}, "")

	// Buffer the command into the Commands channel
	e.Commands <- builtCmd
}

func (e *Endpoint) AddFileQueue(remotePath string, localPath string) {
	e.FileQueue[remotePath] = &FileData{
		LocalPath: localPath,
		ShaHash:   "",
	}
}

func (e *Endpoint) AddUploadQueue(localPath string) {
	e.UploadQueue = append(e.UploadQueue, localPath)
}

func (e *Endpoint) ClearUploadQueue() {
	e.UploadQueue = []string{}
}

func (e *Endpoint) RemoveFileQueue(remotePath string) {
	if _, exists := e.FileQueue[remotePath]; !exists {
		return
	}
	delete(e.FileQueue, remotePath)
}

// verifyBeaconMessage will determine if the initial beacon packet contains the necessary information to continue
// with communications
func verifyBeaconMessage(msgType string, msgArgs []string) error {
	if len(msgArgs) < 3 || msgType != "MSG:" || msgArgs[0] != "BEACON" {
		return errors.New("malformed initial beacon message")
	}
	return nil
}

func (e *Endpoint) processMessage(s *Server, msgType string, msgArgs []string) {
	// If the message is not empty, and the message type is not IMSG or FILE, then log the message to the MsgData slice
	if msgType != "" && msgType != "IMSG:" && msgType != "FILE:" {
		msg := msgType
		for _, s := range msgArgs {
			msg += s
		}
		e.MsgData = append(e.MsgData, msg)
	}
	// If the MsgType is "IMSG" then process it as an immediate message to be printed to the console
	if msgType == "IMSG:" {
		if payload, err := base64.StdEncoding.DecodeString(msgArgs[0]); err != nil {
			fmt.Println("malformed IMSG payload")
		} else {
			// Log the received message into the MsgData slice
			e.MsgData = append(e.MsgData, string(payload))
			// Push the message to the Console's InterMsg channel
			s.Cons.InterMsg <- string(payload)
		}
		// Return as there is no point in performing additional operations
		return
	}

	if msgType == "MSG:" {
		switch strings.ToUpper(msgArgs[0]) {
		case "HOST":
			e.msgHost(s, msgArgs)
			break
		case "PONG":
			e.msgPong()
			break
		case "FILE":
			e.msgFile(msgArgs)
		case "UFILE":
			e.msgUploadFile(msgArgs)
		default:
			fmt.Printf("unsupported message: %s\n", msgType+"\x02"+strings.Join(msgArgs, ""))
		}
		return
	}

	if msgType == "FILE:" {
		if len(msgArgs) < 2 {
			fmt.Println("malformed FILE payload")
			return
		}
		e.handleFileDownload(msgArgs[0], msgArgs[1])
	}
	if msgType == "FILEEND:" {
		if len(msgArgs) < 1 {
			fmt.Println("malformed FILEEND payload", msgArgs)
			return
		}
		e.handleFileDownloadEnd(msgArgs[0])
	}

}

func (e *Endpoint) handleFileDownload(remotePath string, data string) {
	// Check if the server expects this file
	if _, exists := e.FileQueue[remotePath]; !exists {
		fmt.Println("received a FILE payload, but the file was not expected")
		return
	}
	// Get the localPath, fileHash sent by the client, and convert the base64 encoded string into bytes
	localPath := e.FileQueue[remotePath].LocalPath
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		fmt.Println("err decoding file download: ", err)
		return
	}

	// Open a new file handle, create the file if it does not exist.
	file, err := os.OpenFile(localPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)
	defer file.Close()
	if err != nil {
		fmt.Println("err opening file to be written: ", err)
		return
	}
	_, err = file.Write(bytes)
	if err != nil {
		fmt.Println("err writing file: ", err)
		file.Close()
		return
	}
	// Commit to write changes
	file.Sync()
}

func (e *Endpoint) handleFileDownloadEnd(remotePath string) {
	localPath := e.FileQueue[remotePath].LocalPath
	hash := e.FileQueue[remotePath].ShaHash
	file, err := os.OpenFile(localPath, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		fmt.Println("err opening file handle to read file: ", err)
		return
	}
	// Reset file handle
	file.Seek(0, io.SeekStart)
	// Calculate the sha-256 checksum of the written file
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		fmt.Println("err calculating checksum of file hash: ", err)
		return
	}
	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHash != hash {
		fmt.Println("file downloaded, but there is a checksum mismatch L: ", calculatedHash, " R: ", hash)
		return
	}
	e.RemoveFileQueue(remotePath)
}
