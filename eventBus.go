package main

import (
	"sync"
)

// EventBus is a struct to hold all data about the endpoints
// registered on the multiq server. It acts like a bus between
// all the request queues and the client. All requests from the
// client are added to the correct queues by the EventBus
type EventBus struct {
	dataEndpoints map[string]DataEndpoint
	rm            sync.RWMutex
}

// CompleteChan is a channel on which the EventBus will notify
// the http handler function to return and hence close the
// ResponseWriter, thus sending the client the response of the
// forwarded request.
type CompleteChan chan bool

// AddEndpoint to eventbus
func (eb *EventBus) AddEndpoint(de DataEndpoint) {
	eb.rm.Lock()
	eb.dataEndpoints[de.id] = de
	eb.rm.Unlock()
}

// Publish to eventbus
func (eb *EventBus) Publish(id string, data *HTTPRequestData) {
	eb.rm.RLock()
	de := eb.dataEndpoints[id]
	*(de.dataChannel) <- data
	eb.rm.RUnlock()
}
