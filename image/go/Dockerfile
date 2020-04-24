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