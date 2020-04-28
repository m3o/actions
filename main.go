package main

import (
	"fmt"
	"log"
	"os"
	"sync"

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
		getEnv("GITHUB_SHA"),
		getEnv("GITHUB_WORKFLOW"),
	)

	dirs, err := changes.List()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v services have been changed\n", len(dirs))

	var wg sync.WaitGroup
	wg.Add(len(dirs))

	for dir, status := range dirs {
		go func(dir string, status changedetector.Status) {
			// Send the source created/updated/deleted
			// event. This is important since if the
			// source has been deleted, the service must
			// be removed from the runtime however no
			// build event will fire.
			var srcStatus string
			switch status {
			case changedetector.StatusCreated:
				srcStatus = "source_created"
			case changedetector.StatusUpdated:
				srcStatus = "source_updated"
			case changedetector.StatusDeleted:
				srcStatus = "source_deleted"
			}
			events.Create(dir, srcStatus)

			// don't build directories which have been
			// deleted since the source will no longer
			// be there
			if status == changedetector.StatusDeleted {
				fmt.Printf("[%v] Skipping Build\n", dir)
				wg.Done()
				return
			}

			events.Create(dir, "build_started")
			fmt.Printf("[%v] Build Started\n", dir)

			// build the docker image for the directory
			// and push it to the image repository
			if err := builder.Build(dir); err != nil {
				fmt.Printf("[%v] Build Failed: %v\n", dir, err)
				events.Create(dir, "build_failed", err)
			} else {
				fmt.Printf("[%v] Build Finished\n", dir)
				events.Create(dir, "build_finished")
			}

			wg.Done()
		}(dir, status)
	}

	wg.Wait()
}

func getEnv(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		log.Fatalf("Missing %v env var", key)
	}
	return val
}
