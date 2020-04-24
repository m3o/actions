FROM golang:1.13 as builder
WORKDIR /src/action
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app .

FROM alpine:3.10
COPY --from=builder /src/action /action
WORKDIR action
RUN ls -ll
ENTRYPOINT ["./app"]