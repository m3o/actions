package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/jhoonb/archivex"
)

// Action builds docker images
type Action struct {
	cli        *client.Client
	repository string
}

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	ghRepo := os.Getenv("GITHUB_REPOSITORY")
	if len(ghRepo) == 0 {
		panic("Missing GITHUB_REPOSITORY env var")
	}

	repo := "docker.pkg.github.com/" + ghRepo
	a := Action{cli: cli, repository: repo}

	if err := a.BuildAndPush("test"); err != nil {
		panic(err)
	}
}

// BuildAndPush a docker image using the directory provided
func (a *Action) BuildAndPush(dir string) error {
	// e.g. docker.pkg.github.com/micro/services/foobar-api:latest
	tag := fmt.Sprintf("%v/%v:latest", a.repository, strings.ReplaceAll(dir, "/", "-"))

	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     "image/go/Dockerfile",
		Tags:           []string{tag},
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

	buildRsp, err := a.cli.ImageBuild(context.Background(), buildCtx, opt)
	if err != nil {
		return err
	}
	defer buildRsp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	pushRsp, err := a.cli.ImagePush(context.Background(), tag, types.ImagePushOptions{})
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
