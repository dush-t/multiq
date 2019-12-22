package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

type reqPayloadStruct struct {
	EndpointID string `json:"EndpointId"`
}

var mutex sync.RWMutex
var eb = EventBus{dataEndpoints: make(map[string]DataEndpoint), rm: mutex}

func main() {
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/addEndpoint", addEndpointRoute)
	http.HandleFunc("/pushToEndpoint", pushToEndpoint)

	go handleMessages()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

func pushToEndpoint(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	log.Println("Pushing to endpoint")

	var reqPayload reqPayloadStruct
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	decoder = json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(body)))

	decodeErr := decoder.Decode(&reqPayload)
	if decodeErr != nil {
		panic(err)
	}

	endpointID := reqPayload.EndpointID
	var c = make(CompleteChan)
	var httpData = HttpRequestData{res: &w, req: r, completeChan: &c}

	eb.Publish(endpointID, &httpData)

	// Don't close connection till the reverse proxy returns.
	for {
		completed := <-c
		if completed {
			log.Println("Request complete!")
			return
		}
	}

}

func addEndpointRoute(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var body struct {
		ID       string
		Endpoint string
	}

	err := decoder.Decode(&body)
	if err != nil {
		panic(err)
	}
	log.Println(body.ID)

	var dataChan = make(DataChannel)
	de := DataEndpoint{id: body.ID, dataChannel: &dataChan, endpoint: body.Endpoint, active: false}
	log.Println(de)
	eb.AddEndpoint(de)
	go de.Start()
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	// log.Println(*ws)

	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	clients[ws] = true
	log.Println(clients)

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}

		broadcast <- msg
	}
}

func handleMessages() {
	time.Sleep(20 * time.Second)
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
