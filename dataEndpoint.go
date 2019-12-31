package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

// HTTPRequestData stores the data required to push a request to a particular DataEndpoint
// and to return the response of the pushed request back to the client.
type HTTPRequestData struct {
	res          *http.ResponseWriter // Pointer to original ResponseWriter opened in the handler function.
	req          *http.Request        // Pointer to the request that is to be forwarded to the given endpoint
	completeChan *CompleteChan        // Channel on which the EventBus notifies handler function to return response.
}

// DataChannel is the actual 'queue' of web requests. The goroutine spawned by the DataEndpoint
// Start function will be listening on this channel and will forward requests to the corresponding
// url in the DataEndpoint.
type DataChannel chan *HTTPRequestData

// DataEndpoint is like an address. It contains information of where to send information
// for a single endpoint.
type DataEndpoint struct {
	id          string       // Client will use this id to specify on which queue to add requests
	dataChannel *DataChannel // The actual 'queue'
	endpoint    string       // The actual url where the request is to be sent
	active      bool         // True if the DataEndpoint is in the process of forwarding a request
}

// PushTo is the function that will actually forward a request to a url.
func (d *DataEndpoint) PushTo(h *HTTPRequestData) {
	res := h.res
	req := h.req
	compChan := *h.completeChan

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
		return
	}

	// Create and configure new request for https.
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	url := d.endpoint
	proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
	proxyReq.Header = req.Header
	proxyReq.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	httpClient := &http.Client{}
	resp, reqErr := httpClient.Do(proxyReq)
	if reqErr != nil {
		panic(reqErr)
		return
	}
	defer resp.Body.Close()

	// Write the response recieved from the forwarded request into the original
	// response that the http handler func was supposed to return using the original
	// ResponseWriter.
	for h, val := range resp.Header {
		(*res).Header().Set(h, val[0])
	}
	io.Copy(*res, resp.Body)

	// Notify the http handler function to return the response to the client.
	compChan <- true

}

// Start spawns a goroutine that listens to the dataChannel of a
// DataEndpoint. When the channel recieves a *HTTPRequestData, it
// passes the data to the PushTo function.
func (d *DataEndpoint) Start() {
	dataChan := *(d.dataChannel)
	log.Println(d)
	for {
		httpReqData := <-dataChan
		log.Println("Recieved request data!")
		// d.Push(httpReqData.res, httpReqData.req)
		d.PushTo(httpReqData)
	}
}
