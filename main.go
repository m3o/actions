package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/jhoonb/archivex"
)

var (
	testDir  = "test"
	testRepo = "docker.pkg.github.com/ben-toogood/m3o-test/test:latest"
)

func main() {
	fmt.Printf("CI: %v", os.Getenv("CI"))
	fmt.Printf("HOME: %v", os.Getenv("HOME"))
	fmt.Printf("GITHUB_WORKFLOW: %v", os.Getenv("GITHUB_WORKFLOW"))
	fmt.Printf("GITHUB_RUN_ID: %v", os.Getenv("GITHUB_RUN_ID"))
	fmt.Printf("GITHUB_RUN_NUMBER: %v", os.Getenv("GITHUB_RUN_NUMBER"))
	fmt.Printf("GITHUB_ACTION: %v", os.Getenv("GITHUB_ACTION"))
	fmt.Printf("GITHUB_ACTIONS: %v", os.Getenv("GITHUB_ACTIONS"))
	fmt.Printf("GITHUB_ACTOR: %v", os.Getenv("GITHUB_ACTOR"))
	fmt.Printf("GITHUB_REPOSITORY: %v", os.Getenv("GITHUB_REPOSITORY"))
	fmt.Printf("GITHUB_EVENT_NAME: %v", os.Getenv("GITHUB_EVENT_NAME"))
	fmt.Printf("GITHUB_EVENT_PATH: %v", os.Getenv("GITHUB_EVENT_PATH"))
	fmt.Printf("GITHUB_WORKSPACE: %v", os.Getenv("GITHUB_WORKSPACE"))
	fmt.Printf("GITHUB_SHA: %v", os.Getenv("GITHUB_SHA"))
	fmt.Printf("GITHUB_REF: %v", os.Getenv("GITHUB_REF"))
	fmt.Printf("GITHUB_HEAD_REF: %v", os.Getenv("GITHUB_HEAD_REF"))
	fmt.Printf("GITHUB_BASE_REF: %v", os.Getenv("GITHUB_BASE_REF"))

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	if err := buildService(cli, testDir); err != nil {
		panic(err)
	}
}

func buildService(cli *client.Client, dir string) error {
	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     "image/go/Dockerfile",
		Tags:           []string{testRepo},
		BuildArgs: map[string]*string{
			"service_dir": &dir,
		},
	}

	tar := new(archivex.TarFile)
	tar.Create("/tmp/test.tar")
	tar.AddAll("image", true)
	tar.AddAll(dir, true)
	tar.Close()

	buildCtx, err := os.Open("/tmp/test.tar")
	if err != nil {
		return err
	}
	defer buildCtx.Close()

	buildRsp, err := cli.ImageBuild(context.Background(), buildCtx, opt)
	if err != nil {
		return err
	}
	defer buildRsp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	pushOpts := types.ImagePushOptions{}
	pushRsp, err := cli.ImagePush(context.Background(), testRepo, pushOpts)
	if err != nil {
		return err
	}
	defer pushRsp.Close()

	err = jsonmessage.DisplayJSONMessagesStream(pushRsp, os.Stdout, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	return nil
}
