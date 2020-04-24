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
	"github.com/google/go-github/v30/github"
	"github.com/jhoonb/archivex"
)

const (
	githubRegistry = "docker.pkg.github.com"

	// Dockerfile template to use. GitHub actions only copies the entrypoint
	// into the docker container it starts when running an action, so we have
	// to write this file on each run...
	dockerfile = `
FROM golang:1.13 as builder

# Install Dumb Init
RUN git clone https://github.com/Yelp/dumb-init.git
WORKDIR ./dumb-init
RUN make build
WORKDIR ..

# Copy the service
ARG service_dir
COPY $service_dir service
WORKDIR service

# Build the service
RUN go get -d -v
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app .

# A distroless container image with some basics like SSL certificates
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static
COPY --from=builder /go/service/app /service
COPY --from=builder /go/dumb-init/dumb-init /dumb-init
ENTRYPOINT ["dumb-init", "./service"]
`
)

// Action builds docker images
type Action struct {
	docker      *client.Client
	client      *github.Client
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

	if err := a.build(dir, tag); err != nil {
		return err
	}

	return a.push(tag)
}

// build a docker image using the directory provided
func (a *Action) build(dir, tag string) error {
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
	tar.Add("Dockerfile", strings.NewReader(dockerfile), nil)
	tar.AddAll(dir, true)
	tar.Close()

	buildCtx, err := os.Open(ctxPath)
	if err != nil {
		return err
	}
	defer buildCtx.Close()

	buildRsp, err := a.docker.ImageBuild(context.Background(), buildCtx, opt)
	if err != nil {
		return err
	}
	defer buildRsp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	return jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
}

// push a tagged image to the image repo
func (a *Action) push(tag string) error {
	pushOpts := types.ImagePushOptions{RegistryAuth: a.registryCreds()}
	pushRsp, err := a.docker.ImagePush(context.Background(), tag, pushOpts)
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
