package subscriber

import (
	"context"
	log "github.com/micro/go-micro/v2/logger"

	test "test/proto/test"
)

type Test struct{}

func (e *Test) Handle(ctx context.Context, msg *test.Message) error {
	log.Info("Handler Received message: ", msg.Say)
	return nil
}

func Handler(ctx context.Context, msg *test.Message) error {
	log.Info("Function Received message: ", msg.Say)
	return nil
}
