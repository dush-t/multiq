package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type HttpRequestData struct {
	res          *http.ResponseWriter
	req          *http.Request
	completeChan *CompleteChan
}

// DataChannel channel to send pub/sub messages
type DataChannel chan *HttpRequestData

// DataEndpoint to forward data to server from the channel
type DataEndpoint struct {
	id          string
	dataChannel *DataChannel
	endpoint    string
	active      bool
}

// Push data to the endpoint and lock till it gets a response.
func (d *DataEndpoint) Push(res *http.ResponseWriter, req *http.Request) {
	d.active = true

	url, _ := url.Parse(d.endpoint)
	proxy := httputil.NewSingleHostReverseProxy(url)
	req.URL.Host = url.Host
	log.Println(req.URL.Host)
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host
	log.Printf("Sending request to %v \n", url)
	proxy.ServeHTTP(*res, req)

	d.active = false
}

// PushTo specific endpoint
func (d *DataEndpoint) PushTo(h *HttpRequestData) {
	res := h.res
	req := h.req
	compChan := *h.completeChan

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
		return
	}

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

	for h, val := range resp.Header {
		(*res).Header().Set(h, val[0])
	}
	io.Copy(*res, resp.Body)
	compChan <- true

}

// Start the dataendpoint
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
