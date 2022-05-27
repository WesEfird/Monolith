# Monolith Server
A console based C2 (Command & Control) platform written in Go.
- All network communications wrapped in TLS.
- Asynchronously sends and receives network data to endpoints.
- Interactive console prompt with suggestions and auto-complete.
- Remotely execute commands on endpoints and receive command output.
- Endpoints will beacon the server at configurable intervals.

## TODO
- [x] Improve tracking of deployed endpoints. Possibly implement a database.
- [x] Implement authentication between the server and endpoints.
- [x] Implement the building of endpoint binaries from the server console.
- [x] Implement file transfers between the server and endpoints.
- [ ] Implement DNS as a transport method for communicating with endpoints.
- [ ] Support ability for a live shell session to an endpoint. (Currently only pseudo-live)
- [ ] Write an endpoint in C. (SSPI is a nightmare)

## Build
### Server
Targeted and tested with Go 1.18.1

[go-prompt](https://github.com/c-bata/go-prompt) is a required dependency.

`go build github.com/WesEfird/Monolith`

`./Monolith`
### Client
The client can be built using Monolith.
Once the Monolith binary has been built and ran, use the `BuildEndpoint` command in the Monolith console.

`BuildEndpoint <EndpointID> <Server IP> <Server Port> <Auth String>`

The endpoint binary will be placed in the Monolith/bin directory, and will be compiled with the settings provided.

A manual build can be achieved by navigating to the Monolith/res/MonolithClient directory and running a go build command.

`go build -o ../../bin/client.exe -ldflags=-s -w 
-X'main.EndpointID=<EndpointID>' -X 'main.IpAddr=<Server IP>' -X 'main.Port=<Server Port>' -X 'main.AuthString=<Auth String>'`

## Disclaimer
The purpose of this tool is to allow the study of adversary tactics and enable security researchers to have access to a live C2 implementation;
this implementation is to be used as a tool for educational purposes, and for the development of defensive rules, tactics, 
and techniques. This program is intended to only be used in environments that the user owns and controls, or in 
environments where the user has explicit permission to run offensive security tools. The user must adhere to all laws 
in their jurisdiction and must conduct themselves ethically when using this tool.