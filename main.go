package main

import (
	"flag"
	"log"
	"os"
)

func main() {

	projectID := flag.String("projectID", "", "List Queries running under project")
	port := flag.Int("port", 8080, "http port to listen on")
	sessionSecret := flag.String("sessionSecret", "itsy bitsy spider climbed up the water spout", "Some nonsensical 'secure' string")

	flag.Parse()

	if *projectID == "" {
		// try to get projectID from env
		*projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if *projectID == "" {
			log.Fatalf("projectID can't be empty")
		}
	}
	service := NewBQService(*projectID)
	log.Printf("Running BaqMan for project: %v on port %d", *projectID, *port)
	RunServer(service, *port, *sessionSecret)
}
