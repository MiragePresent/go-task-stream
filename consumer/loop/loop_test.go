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
	sub := &testSub[string]{
		msgs: make(chan taskstream.Message[string], 1),
		nackFn: func(id string, reason error) error {
			_ = id
			if !errors.Is(reason, wantErr) {
				t.Fatalf("sub.Nack() reason = %v, want %v", reason, wantErr)
			}
			return taskstream.ErrNotSupported
		},
	}
	sub.msgs <- taskstream.Message[string]{ID: "2", Data: "bad"}
	close(sub.msgs)

	err := Consume(context.Background(), sub, func(ctx context.Context, msg taskstream.Message[string]) error {
		_ = ctx
		_ = msg
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Consume() error = %v, want %v", err, wantErr)
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

