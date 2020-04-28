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
		len(os.Getenv("INPUT_DEBUG_MODE")) > 0,
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
		// can't build the image since the source no longer exists
		if status == changedetector.StatusDeleted {
			continue
		}

		events.Create(dir, "build_started")
		fmt.Printf("[%v] Build Starting\n", dir)

		if err := builder.Build(dir); err != nil {
			fmt.Printf("[%v] Build Failed: %v\n", dir, err)
			events.Create(dir, "build_failed", err)
		} else {
			fmt.Printf("[%v] Build Finished\n", dir)
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
