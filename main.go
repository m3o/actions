package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/google/go-github/v30/github"
)

func main() {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	a := Action{
		docker:      docker,
		client:      github.NewClient(nil),
		apiKey:      getEnv("INPUT_API_KEY"),
		githubRepo:  getEnv("GITHUB_REPOSITORY"),
		githubOwner: getEnv("GITHUB_REPOSITORY_OWNER"),
		githubToken: getEnv("INPUT_GITHUB_TOKEN"),
	}

	dirs, err := a.ListChangedDirectories(getEnv("GITHUB_SHA"))
	if err != nil {
		panic(err)
	}

	for dir, status := range dirs {
		fmt.Printf("Processing %v status for %v dir\n", status, dir)

		// can't build the image since the source no longer exists
		if status == ServiceStatusDeleted {
			continue
		}

		if err := a.BuildAndPush(dir); err != nil {
			panic(err)
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
