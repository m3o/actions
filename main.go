package main

import (
	"log"
	"os"

	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	a := Action{
		cli:         cli,
		apiKey:      getEnv("INPUT_API_KEY"),
		githubRepo:  getEnv("GITHUB_REPOSITORY"),
		githubOwner: getEnv("GITHUB_REPOSITORY_OWNER"),
		githubToken: getEnv("INPUT_GITHUB_TOKEN"),
	}

	if err := a.BuildAndPush("test"); err != nil {
		panic(err)
	}
}

func getEnv(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		log.Fatalf("Missing %v env var", key)
	}
	return val
}
