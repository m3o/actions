package main

import (
	"fmt"
	"log"
	"os"

	"github.com/micro/actions/builder"
	"github.com/micro/actions/changedetector"
	"github.com/micro/actions/events"
)

func main() {
	builder := builder.New(
		getEnv("INPUT_GITHUB_TOKEN"),
		getEnv("GITHUB_REPOSITORY"),
		getEnv("GITHUB_REPOSITORY_OWNER"),
	)

	changes := changedetector.New(
		getEnv("INPUT_GITHUB_TOKEN"),
		getEnv("GITHUB_REPOSITORY"),
		getEnv("GITHUB_REPOSITORY_OWNER"),
		getEnv("GITHUB_SHA"),
	)

	events := events.New(
		getEnv("INPUT_CLIENT_ID"),
		getEnv("INPUT_CLIENT_SECRET"),
	)

	dirs, err := changes.List()
	if err != nil {
		panic(err)
	}

	for dir, status := range dirs {
		fmt.Printf("Processing %v status for %v dir\n", status, dir)

		// can't build the image since the source no longer exists
		if status == changedetector.StatusDeleted {
			continue
		}

		events.Create(dir, "build_started")
		if err := builder.Build(dir); err != nil {
			events.Create(dir, "build_failed", err)
		} else {
			events.Create(dir, "build_finished")
		}
	}
}

func getEnv(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		log.Fatalf("Missing %v env var", key)
	}
	return val
}
