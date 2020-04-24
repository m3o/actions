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

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	ghRepo := os.Getenv("GITHUB_REPOSITORY")
	if len(ghRepo) == 0 {
		panic("Missing GITHUB_REPOSITORY env var")
	}

	ghToken := os.Getenv("INPUT_GITHUB_TOKEN")
	if len(ghToken) == 0 {
		panic("Missing INPUT_GITHUB_TOKEN env var")
	}

	apiKey := os.Getenv("INPUT_API_KEY")
	if len(apiKey) == 0 {
		panic("Missing INPUT_API_KEY env var")
	}

	a := Action{
		cli:         cli,
		githubRepo:  ghRepo,
		githubToken: ghToken,
	}

	if err := a.BuildAndPush("test"); err != nil {
		panic(err)
	}
}

// Action builds docker images
type Action struct {
	cli         *client.Client
	githubRepo  string
	githubToken string
}

// BuildAndPush a docker image using the directory provided
func (a *Action) BuildAndPush(dir string) error {
	// e.g. foobar/api => foobar-api
	formattedDir := strings.ReplaceAll(dir, "/", "-")

	// e.g. docker.pkg.github.com/micro/services/foobar-api:latest
	tag := fmt.Sprintf("%v/%v/%v:latest", githubRegistry, a.githubRepo, formattedDir)

	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     "Dockerfile",
		Tags:           []string{tag},
		BuildArgs: map[string]*string{
			"service_dir": &dir,
		},
	}

	ctxPath := fmt.Sprintf("/tmp/%v.tar", formattedDir)
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
	err = jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	pushOpts := types.ImagePushOptions{RegistryAuth: a.registryCreds()}
	pushRsp, err := a.cli.ImagePush(context.Background(), tag, pushOpts)
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

func (a *Action) registryCreds() string {
	creds := map[string]string{
		"password":      a.githubToken,
		"serveraddress": githubRegistry,
	}

	bytes, _ := json.Marshal(creds)
	return base64.StdEncoding.EncodeToString(bytes)
}
