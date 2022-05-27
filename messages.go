package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// msgHost will expect msgArgs to have a length of at least 2.
//
// msgArgs[0] will be the "HOST" message descriptor.
//
// msgArgs[1] will be the hostname of the remote endpoint.
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

// msgPong is used primarily for testing
func (e *Endpoint) msgPong() {
	//fmt.Printf("Received ping from %s\n", e.Conn.RemoteAddr().String())
	return
}

// msgFile will expect msgArgs to have a length of at least 3 arguments.
//
// msgArgs[0] will be the "FILE" message descriptor.
//
// msgArgs[1] will be the remote path of the file, including the filename and extension (if it has one).
//
// msgArgs[2] will be the Sha-256 hash of the file.
func (e *Endpoint) msgFile(msgArgs []string) {
	if len(msgArgs) < 3 {
		fmt.Println("malformed FILE message")
		return
	}
	if _, exists := e.FileQueue[msgArgs[1]]; !exists {
		fmt.Println("received a message to download a file, but the file is not expected: ", msgArgs[1])
		return
	}

	e.FileQueue[msgArgs[1]].ShaHash = msgArgs[2]
}

// msgUploadFile will expect msgArgs to have a length of at least 2 arguments
//
// msgArgs[0] = "FILE" message descriptor
//
// msgArgs[1] = "local file name"
func (e *Endpoint) msgUploadFile(msgArgs []string) {
	var path string
	fileName := msgArgs[1]

	// check if the file is in the uploadQueue, and if so, then set the path var to the filepath found in the uploadQueue
	isInQueue := func() bool {
		for _, value := range e.UploadQueue {
			if filepath.Base(value) == fileName {
				path = value
				return true
			}
		}
		return false
	}()
	// if the file is not in the uploadQueue, then return
	if !isInQueue {
		fmt.Println("received request to upload a file to an endpoint, however, the file was not found in the uploadQueue: ", fileName)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Println("err grabbing handle on file: ", err)
		return
	}
	defer file.Close()
	buf := make([]byte, 4028)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("err in file read operation: ", err)
			return
		}
		if err == io.EOF {
			break
		}
		payload := base64.StdEncoding.EncodeToString(buf[:n])
		e.Conn.Write([]byte("FILE:\x02" + fileName + "\x02" + payload + "\n"))
	}
	e.Conn.Write([]byte("FILEEND:" + "\x02" + fileName + "\n"))
}
