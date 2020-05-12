package builder

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	// Dockerfile template to use. GitHub actions only copies the entrypoint
	// into the docker container it starts when running an action, so we have
	// to write this file on each run...
	dockerfile = `
FROM golang:1.13 as builder

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
ENTRYPOINT ["./service"]
`
)

// New returns an initialized builder
func New(token, repo, owner string, debug bool) *Builder {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &Builder{
		docker:      docker,
		githubRepo:  repo,
		githubOwner: owner,
		githubToken: token,
		debug:       debug,
	}
}

// Builder builds docker images
type Builder struct {
	docker      *client.Client
	githubRepo  string
	githubOwner string
	githubToken string
	debug       bool
}

// Build and pushe a docker image using the directory provided
func (b *Builder) Build(dir string) error {
	// e.g. foobar/api => foobar-api
	formattedDir := strings.ReplaceAll(dir, "/", "-")

	// e.g. docker.pkg.github.com/micro/services/foobar-api:latest
	tag := fmt.Sprintf("%v/%v/%v:latest", githubRegistry, b.githubRepo, formattedDir)

	if err := b.build(dir, tag); err != nil {
		return err
	}

	return b.push(tag)
}

// build a docker image using the directory provided
func (b *Builder) build(dir, tag string) error {
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

	buildRsp, err := b.docker.ImageBuild(context.Background(), buildCtx, opt)
	if err != nil {
		return err
	}
	defer buildRsp.Body.Close()

	if b.debug {
		termFd, isTerm := term.GetFdInfo(os.Stdout)
		return jsonmessage.DisplayJSONMessagesStream(buildRsp.Body, os.Stdout, termFd, isTerm, nil)
	}

	_, err = ioutil.ReadAll(buildRsp.Body)
	return err
}

// push a tagged image to the image repo
func (b *Builder) push(tag string) error {
	pushOpts := types.ImagePushOptions{RegistryAuth: b.registryCreds()}
	pushRsp, err := b.docker.ImagePush(context.Background(), tag, pushOpts)
	if err != nil {
		return err
	}
	defer pushRsp.Close()

	if b.debug {
		termFd, isTerm := term.GetFdInfo(os.Stdout)
		return jsonmessage.DisplayJSONMessagesStream(pushRsp, os.Stdout, termFd, isTerm, nil)
	}

	_, err = ioutil.ReadAll(pushRsp)
	return err
}

func (b *Builder) registryCreds() string {
	creds := map[string]string{
		"username":      b.githubOwner,
		"password":      b.githubToken,
		"serveraddress": githubRegistry,
	}

	bytes, _ := json.Marshal(creds)
	return base64.StdEncoding.EncodeToString(bytes)
}
