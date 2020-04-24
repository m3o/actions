package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/jhoonb/archivex"
)

const (
	githubRegistry = "docker.pkg.github.com"
)

// Action builds docker images
type Action struct {
	cli         *client.Client
	apiKey      string
	githubRepo  string
	githubOwner string
	githubToken string
}

// BuildAndPush a docker image using the directory provided
func (a *Action) BuildAndPush(dir string) error {
	// e.g. foobar/api => foobar-api
	formattedDir := strings.ReplaceAll(dir, "/", "-")

	// e.g. docker.pkg.github.com/micro/services/foobar-api:latest
	tag := fmt.Sprintf("%v/%v/%v:latest", githubRegistry, a.githubRepo, formattedDir)

	if err := a.Build(dir, tag); err != nil {
		return err
	}

	return a.Push(tag)
}

// Build a docker image using the directory provided
func (a *Action) Build(dir, tag string) error {
	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     "Dockerfile",
		Tags:           []string{tag},
		BuildArgs: map[string]*string{
			"service_dir": &dir,
		},
	}

	ctxPath := fmt.Sprintf("/tmp/%v.tar", strings.ReplaceAll(dir, "/", "-"))
	tar := new(archivex.TarFile)
	tar.Create(ctxPath)
	tar.Add("Dockerfile", strings.NewReader(Dockerfile), nil)
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
	return jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
}

// Push a tagged image to the image repo
func (a *Action) Push(tag string) error {
	pushOpts := types.ImagePushOptions{RegistryAuth: a.registryCreds()}
	pushRsp, err := a.cli.ImagePush(context.Background(), tag, pushOpts)
	if err != nil {
		return err
	}
	defer pushRsp.Close()

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	return jsonmessage.DisplayJSONMessagesStream(pushRsp, os.Stdout, termFd, isTerm, nil)
}

func (a *Action) registryCreds() string {
	creds := map[string]string{
		"username":      a.githubOwner,
		"password":      a.githubToken,
		"serveraddress": githubRegistry,
	}

	bytes, _ := json.Marshal(creds)
	fmt.Println(string(bytes))
	return base64.StdEncoding.EncodeToString(bytes)
}
