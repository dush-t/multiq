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

// DynamicHandler is a custom handler in which one can
// use regex in place of hardcoded urls.
type DynamicHandler struct {
	routes []*route
}

type routeQueueData struct {
	open                 bool
	lastRequestTimestamp time.Time
	queue                *DataChannel
	completeChan         *CompleteChan
}

type routeMap map[string]*routeQueueData

// RouteMap stores information about which queues on routes
var RouteMap = make(routeMap)

// Handler function to implement the http.Handler interface
func (d *DynamicHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	d.routes = append(d.routes, &route{pattern, handler})
}

// HandleFunc to implement the http.Handler interface
func (d *DynamicHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	// queue := make(DataChannel)
	// completeChan := make(CompleteChan)
	// data := routeQueueData{open: true, lastRequestTimestamp: time.Now(), queue: &queue, completeChan: &completeChan}
	// RouteMap[pattern] = &data

	d.routes = append(d.routes, &route{pattern, http.HandlerFunc(handler)})
}

// ServeHTTP to implement the http.Handler interface
func (d *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range d.routes {
		if route.pattern.MatchString(r.URL.Path) {
			_, exists := RouteMap[r.URL.Path]
			if !exists {
				queue := make(DataChannel)
				completeChan := make(CompleteChan)
				data := routeQueueData{open: true, lastRequestTimestamp: time.Now(), queue: &queue, completeChan: &completeChan}
				RouteMap[r.URL.Path] = &data
				go queueListener(&data)
			}

			data := RouteMap[r.URL.Path]
			requestData := HTTPRequestData{res: &w, req: r, completeChan: data.completeChan}
			queue := *(data.queue)
			queue <- &requestData

			completed := <-*(data.completeChan)
			if completed == true {
				return
			}
		}
	}
	// simply forward the request without queueing if the URL is not specified in the config file.
	log.Println("Recieved request to unlisted path '" + r.URL.Path + "', forwarding without queueing.")
	forwardRequest(&w, r, nil)
}

func queueListener(r *routeQueueData) {
	q := r.queue
	c := r.completeChan
	for {
		reqData := <-*(q)
		forwardRequest((*reqData).res, (*reqData).req, c)
	}
}

func forwardRequest(res *http.ResponseWriter, req *http.Request, c *CompleteChan) {
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
