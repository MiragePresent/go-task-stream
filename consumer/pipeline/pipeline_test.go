package pipeline

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

func TestConsumePipelineSuccess(t *testing.T) {
	t.Parallel()

	sub := &testSub[int]{
		msgs: make(chan taskstream.Message[int], 1),
	}
	sub.msgs <- taskstream.Message[int]{ID: "1", Data: 2}
	close(sub.msgs)

	var routed int
	err := Consume(context.Background(), sub, []Step[int]{
		func(ctx context.Context, msg taskstream.Message[int]) (taskstream.Message[int], error) {
			_ = ctx
			msg.Data++
			return msg, nil
		},
		func(ctx context.Context, msg taskstream.Message[int]) (taskstream.Message[int], error) {
			_ = ctx
			msg.Data = msg.Data * 3
			return msg, nil
		},
	}, func(ctx context.Context, msg taskstream.Message[int]) error {
		_ = ctx
		routed = msg.Data
		return nil
	})
	if err != nil {
		t.Fatalf("Consume() error = %v, want nil", err)
	}
	if routed != 9 {
		t.Fatalf("router value = %d, want %d", routed, 9)
	}
}

func TestConsumeStepErrorNackNotSupported(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("step failed")
	sub := &testSub[int]{
		msgs: make(chan taskstream.Message[int], 1),
		nackFn: func(id string, reason error) error {
			_ = id
			if !errors.Is(reason, wantErr) {
				t.Fatalf("sub.Nack() reason = %v, want %v", reason, wantErr)
			}
			return taskstream.ErrNotSupported
		},
	}
	sub.msgs <- taskstream.Message[int]{ID: "2", Data: 1}
	close(sub.msgs)

	err := Consume(context.Background(), sub, []Step[int]{
		func(ctx context.Context, msg taskstream.Message[int]) (taskstream.Message[int], error) {
			_ = ctx
			_ = msg
			return taskstream.Message[int]{}, wantErr
		},
	}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Consume() error = %v, want %v", err, wantErr)
	}
}

func TestConsumeRouterErrorAndAckError(t *testing.T) {
	t.Parallel()

	t.Run("router-error", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("route failed")
		sub := &testSub[int]{
			msgs: make(chan taskstream.Message[int], 1),
			nackFn: func(id string, reason error) error {
				_ = id
				if !errors.Is(reason, wantErr) {
					t.Fatalf("sub.Nack() reason = %v, want %v", reason, wantErr)
				}
				return nil
			},
		}
		sub.msgs <- taskstream.Message[int]{ID: "3", Data: 1}
		close(sub.msgs)

		err := Consume(context.Background(), sub, nil, func(ctx context.Context, msg taskstream.Message[int]) error {
			_ = ctx
			_ = msg
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("Consume() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("ack-error", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("ack failed")
		sub := &testSub[int]{
			msgs: make(chan taskstream.Message[int], 1),
			ackFn: func(id string) error {
				_ = id
				return wantErr
			},
		}
		sub.msgs <- taskstream.Message[int]{ID: "4", Data: 1}
		close(sub.msgs)

		err := Consume(context.Background(), sub, nil, nil)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Consume() error = %v, want %v", err, wantErr)
		}
	})
}

