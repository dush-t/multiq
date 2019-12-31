package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

type reqPayloadStruct struct {
	EndpointID string `json:"EndpointId"`
}

var mutex sync.RWMutex
var eb = EventBus{dataEndpoints: make(map[string]DataEndpoint), rm: mutex}

func main() {
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	http.HandleFunc("/addEndpoint", addEndpointRoute)  // Register an endpoint with the multiq server
	http.HandleFunc("/pushToEndpoint", pushToEndpoint) // Push data to a given DataEndpoint

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

/*
	Takes the http request and adds it to the queue of
	the desired endpoint while keeping the connection
	open till the response is recieved from that endpoint.
*/
func pushToEndpoint(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	// Read request body to get EndpointID
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

	/*
		The server will close the connection and return whatever response has been written
		by the ResponseWriter when this function completes execution. So we make a new channel
		c to pass to the EventBus. When the request has been successfully forwarded and a
		response is recieved, the EventBus will send a true value on this channel and only
		then will the function terminate.
	*/
	var c = make(CompleteChan)
	var httpData = HTTPRequestData{res: &w, req: r, completeChan: &c}

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

/*
	Parse http request to get Endpoint details, register an DataEndpoint with
	those details and spawn a goroutine that listens for requests at this
	DataEndpoint.
*/
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
	eb.AddEndpoint(de)
	go de.Start()
}
