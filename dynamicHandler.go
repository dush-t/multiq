package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"time"
)

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type routeQueueData struct {
	open                 bool
	queue                *DataChannel
	lastRequestTimestamp time.Time
}

type routeMap map[string]*routeQueueData

// DynamicHandler is a custom handler in which one can
// use regex in place of hardcoded urls.
type DynamicHandler struct {
	routes []*route
}

// RouteMap stores information about which queues on routes
var RouteMap = make(routeMap)

// Handler function to implement the http.Handler interface
func (d *DynamicHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	d.routes = append(d.routes, &route{pattern, handler})
}

// HandleFunc to implement the http.Handler interface
func (d *DynamicHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	d.routes = append(d.routes, &route{pattern, http.HandlerFunc(handler)})
}

// ServeHTTP to implement the http.Handler interface
func (d *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rqData, toBeQueued := d.findQueue(r.URL.Path)

	c := make(CompleteChan)
	h := HTTPRequestData{
		res:          &w,
		req:          r,
		completeChan: &c,
	}

	if !toBeQueued {
		log.Println("Recieved request to unlisted path '" + r.URL.Path + "', forwarding without queueing.")
		forwardRequest(&h)
	} else {
		rqData.pushRequestToQueue(&h)
	}

}

// findQueue will see if the urlPath is supposed to be queued. If it is not, it will
// simply return nil and false. If it is, then if it's queue is not created, it will
// create and save the queue data. If the queue is already created, it will simply
// return a pointer to the queue data and true.
func (d *DynamicHandler) findQueue(urlPath string) (*routeQueueData, bool) {
	r, exists := RouteMap[urlPath]
	if !exists {
		for _, route := range d.routes {
			if route.pattern.MatchString(urlPath) {
				queue := make(DataChannel)
				rqData := routeQueueData{
					open:                 true,
					lastRequestTimestamp: time.Now(),
					queue:                &queue,
				}
				RouteMap[urlPath] = &rqData
				exists = true

				go queueListener(&rqData) // Start the listener on this queue
				return &rqData, true
			}
		}
		return nil, false
	}
	return r, exists
}

func (r *routeQueueData) pushRequestToQueue(h *HTTPRequestData) {
	queue := *(r.queue)
	queue <- h
	completed := <-*(h.completeChan)
	if completed == true {
		return
	}
}

func queueListener(r *routeQueueData) {
	q := r.queue
	for {
		reqData := <-*(q)
		forwardRequest(reqData)
	}
}

func forwardRequest(h *HTTPRequestData) {
	res := h.res
	req := h.req
	c := h.completeChan

	url, _ := url.Parse(req.URL.Path)
	proxy := httputil.NewSingleHostReverseProxy(url)
	req.URL.Host = url.Host
	log.Println(req.URL.Host)
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host
	log.Printf("Sending request to %v \n", url)
	proxy.ServeHTTP(*res, req)

	if c != nil {
		log.Println("Successfully forwarded the queued request to", req.URL.Path)
		*c <- true
	}
}
