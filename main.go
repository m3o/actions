package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/m3o/action/builder"
	"github.com/m3o/action/changes"
	"github.com/m3o/action/events"
)

func main() {
	debugMode := len(os.Getenv("INPUT_DEBUG_MODE")) > 0

	builder := builder.New(
		getEnv("INPUT_GITHUB_TOKEN"),
		getEnv("GITHUB_REPOSITORY"),
		getEnv("GITHUB_REPOSITORY_OWNER"),
		debugMode,
	)

	change := changes.New(
		getEnv("INPUT_GITHUB_TOKEN"),
		getEnv("GITHUB_REPOSITORY"),
		getEnv("GITHUB_REPOSITORY_OWNER"),
		getEnv("GITHUB_SHA"),
	)

	events := events.New(
		getEnv("INPUT_CLIENT_ID"),
		getEnv("INPUT_CLIENT_SECRET"),
		getEnv("GITHUB_SHA"),
		getEnv("GITHUB_ACTION"),
	)

	dirs, err := change.List()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v services have been changed\n", len(dirs))

	var wg sync.WaitGroup
	wg.Add(len(dirs))
	hasErrored := false

	for dir, status := range dirs {
		f := func(dir string, status changes.Status) {
			// Send the source created/updated/deleted
			// event. This is important since if the
			// source has been deleted, the service must
			// be removed from the runtime however no
			// build event will fire.
			var srcStatus string
			switch status {
			case changes.StatusCreated:
				srcStatus = "source_created"
			case changes.StatusUpdated:
				srcStatus = "source_updated"
			case changes.StatusDeleted:
				srcStatus = "source_deleted"
			}
			events.Create(dir, srcStatus)

			// don't build directories which have been
			// deleted since the source will no longer
			// be there
			if status == changes.StatusDeleted {
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
				hasErrored = true
			} else {
				fmt.Printf("[%v] Build Finished\n", dir)
				events.Create(dir, "build_finished")
			}

			wg.Done()
		}

		// don't perform concurrently in debug mode since
		// all the logs will be combined and unreadable
		if debugMode {
			f(dir, status)
		} else {
			go f(dir, status)
		}
	}

	wg.Wait()
	if hasErrored {
		os.Exit(1)
	}
}

func getEnv(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		log.Fatalf("Missing %v env var", key)
	}
	return val
}
