package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	functions "github.com/KrishnaIyer/go-vanity-cloud-function"
)

func main() {
	funcframework.RegisterHTTPFunction("/", functions.HandleImport)
	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("LOCAL_PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
