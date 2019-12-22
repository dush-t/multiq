# multiq
A lightweight tool to dynamically create web request queues, powered by Go.

## What it does
Say you have a target server A on which you want to queue requests.
Currently, you can build and run the code to spin up a lightweight server (call it B). Queues (corresponding to url endpoint)
can be registered on B by sending a POST request to a particular endpoint. Once a queue is created, a client
can send a request to B to add a request to the queue. B will eventually forward this request to A and then forward the response
back to the client, as if it originated from A. It's simple proxy stuff.

## Possible Use-Case
This was written to be used in Stock Market Simulation 2020, conducted by BITS-ACM. Requests to endpoints for buying/selling
stocks will be queued so that only one web request can access a stock at a time. This will be done to ensure 100% consistency.
This is just an implementation of Pessimistic Concurrency Control at the network level.

## TO-DO
### For the queue - 
* Add better error handling to make code more robust.
* Add more controls for handling the queues.
* Write a client library (for python and javascript) to interact with the Go server.

### Reverse Proxy server - 
* Write code to spin up a server that directly handles requests from a the client (i.e. use the queues to create a reverse proxy
server with queues on selected URLs)

## Contributions - 
Because of lack of documentation, you probably can't start contributing on your own. Write me an email and I'll explain the code
to you.
