package loop

import (
	"context"
	"errors"
	"testing"

	taskstream "github.com/miragepresent/go-task-stream"
)

type testSub[T any] struct {
	msgs    chan taskstream.Message[T]
	ackFn   func(string) error
	nackFn  func(string, error) error
	closeFn func() error
}

func (s *testSub[T]) Messages() <-chan taskstream.Message[T] { return s.msgs }
func (s *testSub[T]) Ack(messageID string) error {
	if s.ackFn == nil {
		return nil
	}
	return s.ackFn(messageID)
}
func (s *testSub[T]) Nack(messageID string, reason error) error {
	if s.nackFn == nil {
		return nil
	}
	return s.nackFn(messageID, reason)
}
func (s *testSub[T]) Close() error {
	if s.closeFn == nil {
		return nil
	}
	return s.closeFn()
}

func TestConsumeSuccessAckNotSupported(t *testing.T) {
	t.Parallel()

	sub := &testSub[string]{
		msgs: make(chan taskstream.Message[string], 1),
		ackFn: func(id string) error {
			_ = id
			return taskstream.ErrNotSupported
		},
	}
	sub.msgs <- taskstream.Message[string]{ID: "1", Data: "ok"}
	close(sub.msgs)

	err := Consume(context.Background(), sub, func(ctx context.Context, msg taskstream.Message[string]) error {
		_ = ctx
		if msg.Data != "ok" {
			t.Fatalf("handler msg.Data = %q, want %q", msg.Data, "ok")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Consume() error = %v, want nil", err)
	}
}

func TestConsumeHandlerErrorNackNotSupported(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("handler failed")
	var seenErr error
	processed := 0
	sub := &testSub[string]{
		msgs: make(chan taskstream.Message[string], 2),
		nackFn: func(id string, reason error) error {
			_ = id
			if !errors.Is(reason, wantErr) {
				t.Fatalf("sub.Nack() reason = %v, want %v", reason, wantErr)
			}
			return taskstream.ErrNotSupported
		},
	}
	sub.msgs <- taskstream.Message[string]{ID: "2", Data: "bad"}
	sub.msgs <- taskstream.Message[string]{ID: "3", Data: "good"}
	close(sub.msgs)

	err := ConsumeWithErrors(
		context.Background(),
		sub,
		func(ctx context.Context, msg taskstream.Message[string]) error {
			_ = ctx
			processed++
			if msg.Data == "bad" {
				return wantErr
			}
			return nil
		},
		func(ctx context.Context, msg taskstream.Message[string], err error) {
			_ = ctx
			_ = msg
			seenErr = err
		},
	)
	if err != nil {
		t.Fatalf("ConsumeWithErrors() error = %v, want nil", err)
	}
	if processed != 2 {
		t.Fatalf("processed messages = %d, want %d", processed, 2)
	}
	if !errors.Is(seenErr, wantErr) {
		t.Fatalf("error handler got = %v, want %v", seenErr, wantErr)
	}
}

func TestConsumeAckErrorSurfaced(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("ack transport error")
	sub := &testSub[string]{
		msgs: make(chan taskstream.Message[string], 1),
		ackFn: func(id string) error {
			_ = id
			return wantErr
		},
	}
	sub.msgs <- taskstream.Message[string]{ID: "3", Data: "ok"}
	close(sub.msgs)

	err := Consume(context.Background(), sub, func(ctx context.Context, msg taskstream.Message[string]) error {
		_ = ctx
		_ = msg
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Consume() error = %v, want %v", err, wantErr)
	}
}

func TestConsumeNackErrorSurfaced(t *testing.T) {
	t.Parallel()

	handlerErr := errors.New("handler failed")
	wantErr := errors.New("nack transport error")
	sub := &testSub[string]{
		msgs: make(chan taskstream.Message[string], 1),
		nackFn: func(id string, reason error) error {
			_ = id
			if !errors.Is(reason, handlerErr) {
				t.Fatalf("sub.Nack() reason = %v, want %v", reason, handlerErr)
			}
			return wantErr
		},
	}
	sub.msgs <- taskstream.Message[string]{ID: "4", Data: "bad"}
	close(sub.msgs)

	err := Consume(context.Background(), sub, func(ctx context.Context, msg taskstream.Message[string]) error {
		_ = ctx
		_ = msg
		return handlerErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Consume() error = %v, want %v", err, wantErr)
	}
}
