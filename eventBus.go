package main

import (
	"log"
	"sync"
)

// type DataEvent struct {
// 	Data  string
// 	Topic string
// }

// DataChannelSlice to hold datachannels
type DataChannelSlice []DataChannel

// EventBus struct to handle eventbuses
type EventBus struct {
	dataEndpoints map[string]DataEndpoint
	rm            sync.RWMutex
}

type CompleteChan chan bool

// AddEndpoint to eventbus
func (eb *EventBus) AddEndpoint(de DataEndpoint) {
	eb.rm.Lock()
	eb.dataEndpoints[de.id] = de
	eb.rm.Unlock()
}

// Publish to eventbus
func (eb *EventBus) Publish(id string, data *HttpRequestData) {
	eb.rm.RLock()
	log.Println("Publishing")
	de := eb.dataEndpoints[id]
	*(de.dataChannel) <- data
	log.Println("Data sent to channel")
	eb.rm.RUnlock()
}
