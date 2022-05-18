package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/WesEfird/Monolith/cryptutil"
	"os"
	"strings"
	"sync"
)

type Server struct {
	Cons          *Console
	Commands      chan string
	Endpoints     []*Endpoint
	EndPointMutex sync.Mutex
}

func NewServer() *Server {
	return &Server{
		Commands: make(chan string, 50),
	}
}

func (s *Server) ProcessCommands() {
	for cmd := range s.Commands {
		switch strings.ToUpper(cmd) {
		case "SAVE":
			s.SaveEndpoints()
		}
	}
}

func (s *Server) AddEndpoint(e *Endpoint) {
	// Lock and defer unlock on the EndpointMutex so multiple goRoutines don't attempt to access or modify the Endpoints
	// slice simultaneously
	s.EndPointMutex.Lock()
	defer s.EndPointMutex.Unlock()
	s.Endpoints = append(s.Endpoints, e)
}

// SaveEndpoints will iterate over the servers Endpoints slice and marshall relevant information into json format.
// This information will then be saved to a file
func (s *Server) SaveEndpoints() {
	var allData []*SaveData

	s.EndPointMutex.Lock()
	for _, e := range s.Endpoints {
		allData = append(allData, &SaveData{
			EndpointID: e.EndpointID,
			Hostname:   e.Hostname,
			AuthString: e.AuthString,
		})
	}
	s.EndPointMutex.Unlock()

	data, _ := json.Marshal(allData)
	if err := os.WriteFile("res/endpoints.json", data, 0644); err != nil {
		fmt.Printf("err writing file %s\n", err.Error())
	}
}

func (s *Server) LoadEndpoints() {
	var result []SaveData
	data, err := os.ReadFile("res/endpoints.json")
	if err != nil {
		fmt.Printf("err reading endpoints 'database': %s\n", err.Error())
		return
	}
	json.Unmarshal(data, &result)

	for _, value := range result {
		e := NewEndpoint(s, nil, value.EndpointID)
		e.Hostname = value.Hostname
		e.AuthString = value.AuthString
	}
}

func (s *Server) GetEndpointList() []string {
	var endpointList []string
	for _, value := range s.Endpoints {
		endpointList = append(endpointList, value.EndpointID)
	}
	return endpointList
}

func (s *Server) GetEndpointById(endpointId string) (*Endpoint, error) {
	if len(s.Endpoints) < 1 {
		return nil, errors.New("no endpoints exist in the server endpoint list")
	}
	for _, v := range s.Endpoints {
		if v.EndpointID == endpointId {
			return v, nil
		}
	}
	return nil, errors.New("no endpoint with that endpointId found")
}

func (s *Server) VerifyAuthString(endpointId string, authString string) bool {
	e, err := s.GetEndpointById(endpointId)
	if err != nil {
		fmt.Printf("endpoint not found in database, cannot fetch auth string for: %s\n", endpointId)
		return false
	}
	hash := cryptutil.StringToSha256Hex(authString)
	if hash == e.AuthString {
		return true
	}
	fmt.Printf("invalid auth string received for: %s\n", e.EndpointID)
	return false
}
