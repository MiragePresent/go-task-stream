package taskstream

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOpenStreamInvalidName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stream    string
		wantError error
	}{
		{name: "empty", stream: "", wantError: ErrInvalidStream},
		{name: "whitespace", stream: "   ", wantError: ErrInvalidStream},
		{name: "slash", stream: "orders/new", wantError: ErrInvalidStream},
		{name: "space", stream: "orders new", wantError: ErrInvalidStream},
		{name: "valid", stream: "orders.new-v1", wantError: nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := OpenStream[string](tt.stream)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("OpenStream(%q) error = %v, want %v", tt.stream, err, tt.wantError)
			}
			if tt.wantError != nil && got != nil {
				t.Fatalf("OpenStream(%q) stream = %v, want nil", tt.stream, got)
			}
			if tt.wantError == nil && got == nil {
				t.Fatalf("OpenStream(%q) stream = nil, want non-nil", tt.stream)
			}
		})
	}
}

func TestStreamCloseIdempotentAndPostClose(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase2-close")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase2-close", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close() first error = %v, want nil", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close() second error = %v, want nil", err)
	}

	if _, err := stream.Publish(Message[string]{Data: "x"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("stream.Publish() error = %v, want %v", err, ErrClosed)
	}
	if _, err := stream.Subscribe(); !errors.Is(err, ErrClosed) {
		t.Fatalf("stream.Subscribe() error = %v, want %v", err, ErrClosed)
	}
}

func TestSubscriptionCloseIdempotent(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase2-sub-close")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase2-sub-close", err)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("sub.Close() first error = %v, want nil", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("sub.Close() second error = %v, want nil", err)
	}

	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatalf("sub.Messages() channel open, want closed")
		}
	default:
		t.Fatalf("sub.Messages() channel not closed after sub.Close()")
	}
}

func TestStreamCloseClosesActiveSubscriptions(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase2-stream-close-subs")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase2-stream-close-subs", err)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close() error = %v, want nil", err)
	}

	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatalf("sub.Messages() channel open after stream.Close(), want closed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("sub.Messages() channel did not close after stream.Close()")
	}
}

func TestSubscribeOptionsDefaultsAndUnsupported(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase2-options")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase2-options", err)
	}

	subAny, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}
	sub := subAny.(*coreSubscription[string])
	if sub.config.Context == nil {
		t.Fatalf("stream.Subscribe() context = nil, want non-nil default")
	}
	if sub.config.StartAt != "" {
		t.Fatalf("stream.Subscribe() startAt = %q, want empty default", sub.config.StartAt)
	}
	if sub.config.ConsumerGroup != "" {
		t.Fatalf("stream.Subscribe() consumerGroup = %q, want empty default", sub.config.ConsumerGroup)
	}
	if sub.config.BufferSize != 0 {
		t.Fatalf("stream.Subscribe() bufferSize = %d, want 0 default", sub.config.BufferSize)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("sub.Close() error = %v, want nil", err)
	}

	if _, err := stream.Subscribe(WithStartAt("earliest")); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("stream.Subscribe(WithStartAt) error = %v, want %v", err, ErrNotSupported)
	}
	if _, err := stream.Subscribe(WithConsumerGroup("workers")); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("stream.Subscribe(WithConsumerGroup) error = %v, want %v", err, ErrNotSupported)
	}

	bufferedAny, err := stream.Subscribe(WithBufferSize(8))
	if err != nil {
		t.Fatalf("stream.Subscribe(WithBufferSize) error = %v, want nil", err)
	}
	buffered := bufferedAny.(*coreSubscription[string])
	if cap(buffered.messages) != 8 {
		t.Fatalf("cap(sub.Messages()) = %d, want %d", cap(buffered.messages), 8)
	}
	if err := buffered.Close(); err != nil {
		t.Fatalf("buffered.Close() error = %v, want nil", err)
	}
}

func TestSubscribeWithContextCancelClosesSubscription(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase2-context")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase2-context", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := stream.Subscribe(WithContext(ctx))
	if err != nil {
		t.Fatalf("stream.Subscribe(WithContext) error = %v, want nil", err)
	}

	cancel()

	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatalf("sub.Messages() channel open after context cancel, want closed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("sub.Messages() channel did not close after context cancel")
	}
}
