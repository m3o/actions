FROM golang:1.13 as builder
WORKDIR /src/action
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app .

# A distroless container image with some basics like SSL certificates
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static
COPY --from=builder /src/action /action
ENTRYPOINT ["/action/app"]