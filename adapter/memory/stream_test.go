package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	taskstream "github.com/miragepresent/go-task-stream"
)

func TestOpenStreamValidation(t *testing.T) {
	t.Parallel()

	if _, err := OpenStream[string](""); !errors.Is(err, taskstream.ErrInvalidStream) {
		t.Fatalf("OpenStream(empty) error = %v, want %v", err, taskstream.ErrInvalidStream)
	}
	if _, err := OpenStream[string]("orders/new"); !errors.Is(err, taskstream.ErrInvalidStream) {
		t.Fatalf("OpenStream(invalid) error = %v, want %v", err, taskstream.ErrInvalidStream)
	}
	if _, err := OpenStream[string]("orders.new"); err != nil {
		t.Fatalf("OpenStream(valid) error = %v, want nil", err)
	}
}

func TestFanOutPublishSubscribe(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("fanout")
	if err != nil {
		t.Fatalf("OpenStream() error = %v, want nil", err)
	}
	defer stream.Close()

	subA, err := stream.Subscribe(taskstream.WithBufferSize(1))
	if err != nil {
		t.Fatalf("stream.Subscribe(subA) error = %v, want nil", err)
	}
	defer subA.Close()

	subB, err := stream.Subscribe(taskstream.WithBufferSize(1))
	if err != nil {
		t.Fatalf("stream.Subscribe(subB) error = %v, want nil", err)
	}
	defer subB.Close()

	id, err := stream.Publish(taskstream.Message[string]{
		Type: "event",
		Data: "hello",
	})
	if err != nil {
		t.Fatalf("stream.Publish() error = %v, want nil", err)
	}
	if id == "" {
		t.Fatalf("stream.Publish() id = %q, want non-empty", id)
	}

	for i, sub := range []taskstream.Subscription[string]{subA, subB} {
		select {
		case msg := <-sub.Messages():
			if msg.Data != "hello" {
				t.Fatalf("sub[%d] message data = %q, want %q", i, msg.Data, "hello")
			}
			if msg.ID == "" {
				t.Fatalf("sub[%d] message id = %q, want non-empty", i, msg.ID)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub[%d] did not receive published message", i)
		}
	}
}

func TestUnsupportedOptionsAndAckNack(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("unsupported")
	if err != nil {
		t.Fatalf("OpenStream() error = %v, want nil", err)
	}
	defer stream.Close()

	if _, err := stream.Subscribe(taskstream.WithStartAt("earliest")); !errors.Is(err, taskstream.ErrNotSupported) {
		t.Fatalf("stream.Subscribe(WithStartAt) error = %v, want %v", err, taskstream.ErrNotSupported)
	}
	if _, err := stream.Subscribe(taskstream.WithConsumerGroup("workers")); !errors.Is(err, taskstream.ErrNotSupported) {
		t.Fatalf("stream.Subscribe(WithConsumerGroup) error = %v, want %v", err, taskstream.ErrNotSupported)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}
	defer sub.Close()

	if err := sub.Ack("1"); !errors.Is(err, taskstream.ErrNotSupported) {
		t.Fatalf("sub.Ack() error = %v, want %v", err, taskstream.ErrNotSupported)
	}
	if err := sub.Nack("1", errors.New("bad")); !errors.Is(err, taskstream.ErrNotSupported) {
		t.Fatalf("sub.Nack() error = %v, want %v", err, taskstream.ErrNotSupported)
	}
}

func TestCloseSemantics(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("close")
	if err != nil {
		t.Fatalf("OpenStream() error = %v, want nil", err)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close() first error = %v, want nil", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close() second error = %v, want nil", err)
	}

	if _, err := stream.Publish(taskstream.Message[string]{Data: "x"}); !errors.Is(err, taskstream.ErrClosed) {
		t.Fatalf("stream.Publish() error = %v, want %v", err, taskstream.ErrClosed)
	}
	if _, err := stream.Subscribe(); !errors.Is(err, taskstream.ErrClosed) {
		t.Fatalf("stream.Subscribe() error = %v, want %v", err, taskstream.ErrClosed)
	}

	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatalf("sub.Messages() channel open, want closed")
		}
	case <-time.After(time.Second):
		t.Fatalf("sub.Messages() channel not closed after stream.Close()")
	}
}

func TestSubscribeWithContextClosesSubscription(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("ctx-close")
	if err != nil {
		t.Fatalf("OpenStream() error = %v, want nil", err)
	}
	defer stream.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := stream.Subscribe(taskstream.WithContext(ctx))
	if err != nil {
		t.Fatalf("stream.Subscribe(WithContext) error = %v, want nil", err)
	}
	cancel()

	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatalf("sub.Messages() channel open after cancel, want closed")
		}
	case <-time.After(time.Second):
		t.Fatalf("sub.Messages() channel not closed after cancel")
	}
}

func TestConcurrentPublishAndClose(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("concurrent")
	if err != nil {
		t.Fatalf("OpenStream() error = %v, want nil", err)
	}

	sub, err := stream.Subscribe(taskstream.WithBufferSize(2048))
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}
	defer sub.Close()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := stream.Publish(taskstream.Message[string]{Data: "event"})
			if err != nil && !errors.Is(err, taskstream.ErrClosed) {
				t.Errorf("stream.Publish() error = %v, want nil or %v", err, taskstream.ErrClosed)
			}
		}()
	}

	_ = stream.Close()
	wg.Wait()
}

