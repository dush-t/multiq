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


type reqPayloadStruct struct {
	EndpointID string `json:"EndpointId"`
}

var mutex sync.RWMutex
var eb = EventBus{dataEndpoints: make(map[string]DataEndpoint), rm: mutex}

func main() {
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

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