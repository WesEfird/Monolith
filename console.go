package main

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"os"
	"strings"
)

type Console struct {
	PServ            *Server
	SelectedEndpoint *Endpoint
	InterMsg         chan string
	InterMsgTimeout  int
}

type Mode int32

const (
	DefaultMode Mode = iota
	EndpointMode
)

var LivePrefixState struct {
	LivePrefix string
	IsEnable   bool
	ConMode    Mode
}

func NewConsole(s *Server) *Console {
	return &Console{
		PServ:           s,
		InterMsg:        make(chan string, 1),
		InterMsgTimeout: 10,
	}
}

func (cons *Console) DisplayConsole() {
	LivePrefixState.ConMode = DefaultMode
	p := prompt.New(
		cons.executor,
		cons.completer,
		prompt.OptionPrefix(">>> "),
		prompt.OptionLivePrefix(cons.changeLivePrefix),
	)
	p.Run()
}

func (cons *Console) changeLivePrefix() (string, bool) {
	switch LivePrefixState.ConMode {
	case DefaultMode:
		LivePrefixState.LivePrefix = ">>> "
		LivePrefixState.IsEnable = false
	case EndpointMode:
		LivePrefixState.LivePrefix = cons.SelectedEndpoint.EndpointID + "> "
		LivePrefixState.IsEnable = true
	}
	return LivePrefixState.LivePrefix, LivePrefixState.IsEnable
}

func (cons *Console) executor(in string) {
	cmd, args := splitArgs(in)

	if cmd == "" {
		return
	}

	// DefaultMode is the root starting mode
	if LivePrefixState.ConMode == DefaultMode {
		switch cmd {
		case "endpoint":
			cons.cmdEndpoint(args)
		case "listendpoints":
			cons.cmdListEndpoints()
		case "gencert":
			cons.cmdGenCert(args)
		case "buildendpoint":
			cons.cmdBuildEndpoint(args)
		case "exit":
			fmt.Println("server shutting down and program closing...")
			os.Exit(0)
		default:
			fmt.Printf("unknown command: %s\n", cmd)
			return
		}
		// Return so other ConsoleModes aren't processed
		return
	}

	// An endpoint has been selected
	if LivePrefixState.ConMode == EndpointMode {
		switch cmd {
		case "menu":
			LivePrefixState.ConMode = DefaultMode
		case "ping":
			cons.SelectedEndpoint.BufferCommand("PING")
		case "log":
			fmt.Println(cons.SelectedEndpoint.MsgData)
		case "exec":
			cons.cmdExec(args)
		case "set":
			cons.cmdSet(args)
		default:
			fmt.Printf("unkown command: %s\n", cmd)
			return
		}
	}
}

func (cons *Console) completer(in prompt.Document) []prompt.Suggest {
	var sug []prompt.Suggest
	cmd, args := splitArgs(in.TextBeforeCursor())

	if len(args) > 0 {
		return prompt.FilterHasPrefix(cons.getOptions(cmd, args), in.GetWordBeforeCursor(), true)
	}

	switch LivePrefixState.ConMode {
	case DefaultMode:
		sug = []prompt.Suggest{
			{Text: "Endpoint", Description: "Select endpoint by endpointID or IP."},
			{Text: "ListEndpoints", Description: "Returns a list of Endpoints that have beaconed the server."},
			{Text: "BuildEndpoint", Description: "Builds an endpoint with a specified EndpointID"},
			{Text: "GenCert", Description: "Generate certificate and private key for TLS communication."},
			{Text: "Exit", Description: "Shutdown the server and exit the program."},
		}
	case EndpointMode:
		sug = []prompt.Suggest{
			{Text: "Menu", Description: "Exit endpoint mode and return to the root menu."},
			{Text: "Ping", Description: "Send a ping command to the endpoint."},
			{Text: "Set", Description: "Tell the endpoint to set certain options."},
			{Text: "Log", Description: "Displays a list of all message data received from this endpoint."},
			{Text: "Exec", Description: "Execute a command on the remote endpoint."},
		}
	}

	return prompt.FilterHasPrefix(sug, in.GetWordBeforeCursor(), true)
}

func (cons *Console) getOptions(cmd string, args []string) []prompt.Suggest {
	var sug []prompt.Suggest
	switch strings.ToLower(cmd) {
	case "endpoint":
		for _, value := range cons.PServ.GetEndpointList() {
			e, _ := cons.PServ.GetEndpointById(value)
			sug = append(sug, prompt.Suggest{Text: value, Description: e.Hostname})
		}
	case "set":
		if LivePrefixState.ConMode == EndpointMode {
			sug = []prompt.Suggest{
				{Text: "BeaconRate", Description: "Rate in seconds at which the endpoint will beacon the server"},
			}
		}
	default:
		sug = []prompt.Suggest{}
	}
	return sug
}

// splitArgs will take in console input and return the root command and it's arguments as separate variables
func splitArgs(in string) (string, []string) {
	allArgs := strings.Split(in, " ")
	cmd := strings.ToLower(allArgs[0])
	var args []string
	if len(allArgs) > 1 {
		args = allArgs[1:]
	} else {
		args = []string{}
	}

	return cmd, args
}
