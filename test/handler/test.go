package handler

import (
	"context"

	log "github.com/micro/go-micro/v2/logger"

	test "test/proto/test"
)

type Test struct{}

// Call is a single request handler called via client.Call or the generated client code
func (e *Test) Call(ctx context.Context, req *test.Request, rsp *test.Response) error {
	log.Info("Received Test.Call request")
	rsp.Msg = "Hello " + req.Name
	return nil
}

// Stream is a server side stream handler called via client.Stream or the generated client code
func (e *Test) Stream(ctx context.Context, req *test.StreamingRequest, stream test.Test_StreamStream) error {
	log.Infof("Received Test.Stream request with count: %d", req.Count)

	for i := 0; i < int(req.Count); i++ {
		log.Infof("Responding: %d", i)
		if err := stream.Send(&test.StreamingResponse{
			Count: int64(i),
		}); err != nil {
			return err
		}
	}

	return nil
}

// PingPong is a bidirectional stream handler called via client.Stream or the generated client code
func (e *Test) PingPong(ctx context.Context, stream test.Test_PingPongStream) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		log.Infof("Got ping %v", req.Stroke)
		if err := stream.Send(&test.Pong{Stroke: req.Stroke}); err != nil {
			return err
		}
	}
}
