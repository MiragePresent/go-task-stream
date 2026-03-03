package taskstream_test

import (
	"errors"
	"fmt"

	taskstream "github.com/miragepresent/go-task-stream"
)

func ExampleOpenStream() {
	type Action struct {
		Name string
	}

	stream, err := taskstream.OpenStream[Action]("actions")
	if err != nil {
		fmt.Println("open error:", err)
		return
	}
	defer stream.Close()

	sub, err := stream.Subscribe(taskstream.WithBufferSize(16))
	if err != nil {
		fmt.Println("subscribe error:", err)
		return
	}
	defer sub.Close()

	_, err = stream.Publish(taskstream.Message[Action]{
		ID:   "1",
		Type: "action",
		Data: Action{Name: "RunJob"},
	})

	fmt.Println("subscription-open", sub.Messages() != nil)
	fmt.Println("publish-not-supported", errors.Is(err, taskstream.ErrNotSupported))

	// Output:
	// subscription-open true
	// publish-not-supported true
}

func Example_withUnsupportedOptions() {
	stream, err := taskstream.OpenStream[string]("orders")
	if err != nil {
		fmt.Println("open error:", err)
		return
	}
	defer stream.Close()

	_, err = stream.Subscribe(taskstream.WithConsumerGroup("workers"))
	fmt.Println("group-not-supported", errors.Is(err, taskstream.ErrNotSupported))

	_, err = stream.Subscribe(taskstream.WithStartAt("earliest"))
	fmt.Println("start-not-supported", errors.Is(err, taskstream.ErrNotSupported))

	// Output:
	// group-not-supported true
	// start-not-supported true
}
