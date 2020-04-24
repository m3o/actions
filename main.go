package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	// debugging
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		})
	if err != nil {
		log.Println(err)
	}

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
	// e.g. foobar/api => foobar-api
	formattedDir := strings.ReplaceAll(dir, "/", "-")

	// e.g. docker.pkg.github.com/micro/services/foobar-api:latest
	tag := fmt.Sprintf("%v/%v:latest", a.repository, formattedDir)

	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     "image/go/Dockerfile",
		Tags:           []string{tag},
		BuildArgs: map[string]*string{
			"service_dir": &dir,
		},
	}

	ctxPath := fmt.Sprintf("/tmp/%v.tar", formattedDir)
	tar := new(archivex.TarFile)
	tar.Create(ctxPath)
	tar.AddAll("image", true)
	tar.AddAll(dir, true)
	tar.Close()

	buildCtx, err := os.Open(ctxPath)
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
