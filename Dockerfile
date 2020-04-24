FROM golang:1.13 as builder
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o build .

# A distroless container image with some basics like SSL certificates
# https://github.com/GoogleContainerTools/distroless
FROM alpine
# FROM gcr.io/distroless/static
COPY --from=builder app ./
ENTRYPOINT ["/build"]