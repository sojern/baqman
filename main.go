package main

import (
	"flag"
	"log"
)

func main() {

	projectID := flag.String("projectID", "sojern-bigquery", "List Queries running under project")
	port := flag.Int("port", 8080, "http port to listen on")
	sessionSecret := flag.String("sessionSecret", "itsy bitsy spider climbed up the water spout", "Some nonsensical 'secure' string")

	flag.Parse()

	service := NewBQService(*projectID)
	log.Printf("Running BaqMan for project: %v on port %d", *projectID, *port)
	RunServer(service, *port, *sessionSecret)
}
