package taskstream

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestCoreContractOperations(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase3-contract")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase3-contract", err)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = sub.Close()
		_ = stream.Close()
	})

	tests := []struct {
		name      string
		call      func() error
		wantError error
	}{
		{
			name: "publish-not-supported",
			call: func() error {
				_, err := stream.Publish(Message[string]{Data: "payload"})
				return err
			},
			wantError: ErrNotSupported,
		},
		{
			name: "ack-not-supported",
			call: func() error {
				return sub.Ack("1")
			},
			wantError: ErrNotSupported,
		},
		{
			name: "nack-not-supported",
			call: func() error {
				return sub.Nack("1", errors.New("failed"))
			},
			wantError: ErrNotSupported,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("%s error = %v, want %v", tt.name, err, tt.wantError)
			}
		})
	}
}

func TestWrappedErrorSentinelChecks(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase3-wrapped")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase3-wrapped", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	if _, err := stream.Publish(Message[string]{Data: "x"}); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("stream.Publish() error = %v, want sentinel %v", err, ErrNotSupported)
	}

	sub, err := stream.Subscribe()
	if err != nil {
		t.Fatalf("stream.Subscribe() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	if err := sub.Ack("ack-1"); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("sub.Ack() error = %v, want sentinel %v", err, ErrNotSupported)
	}
	if err := sub.Nack("nack-1", errors.New("reason")); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("sub.Nack() error = %v, want sentinel %v", err, ErrNotSupported)
	}

	if _, err := OpenStream[string]("bad/name"); !errors.Is(err, ErrInvalidStream) {
		t.Fatalf("OpenStream(invalid) error = %v, want sentinel %v", err, ErrInvalidStream)
	}

	errWithWrap := fmt.Errorf("wrap: %w", ErrClosed)
	if !errors.Is(errWithWrap, ErrClosed) {
		t.Fatalf("errors.Is(wrap ErrClosed) = false, want true")
	}
}

func TestConcurrentSubscribeAndClose(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase3-concurrent-subscribe-close")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase3-concurrent-subscribe-close", err)
	}

	var wg sync.WaitGroup
	var closeWG sync.WaitGroup

	closeWG.Add(1)
	go func() {
		defer closeWG.Done()
		_ = stream.Close()
	}()

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub, err := stream.Subscribe()
			if err == nil {
				_ = sub.Close()
				return
			}
			if !errors.Is(err, ErrClosed) {
				t.Errorf("stream.Subscribe() error = %v, want nil or %v", err, ErrClosed)
			}
		}()
	}

	wg.Wait()
	closeWG.Wait()
}

func TestConcurrentPublishAndClose(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase3-concurrent-publish-close")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase3-concurrent-publish-close", err)
	}

	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := stream.Publish(Message[string]{ID: fmt.Sprintf("%d", i), Data: "event"})
			if err == nil {
				t.Errorf("stream.Publish() error = nil, want non-nil")
				return
			}
			if !errors.Is(err, ErrNotSupported) && !errors.Is(err, ErrClosed) {
				t.Errorf("stream.Publish() error = %v, want %v or %v", err, ErrNotSupported, ErrClosed)
			}
		}(i)
	}

	_ = stream.Close()
	wg.Wait()
}

func TestConcurrentSubscriptionCloseWithStreamClose(t *testing.T) {
	t.Parallel()

	stream, err := OpenStream[string]("phase3-concurrent-sub-close")
	if err != nil {
		t.Fatalf("OpenStream(%q) error = %v, want nil", "phase3-concurrent-sub-close", err)
	}

	subs := make([]Subscription[string], 0, 100)
	for i := 0; i < 100; i++ {
		sub, err := stream.Subscribe()
		if err != nil {
			t.Fatalf("stream.Subscribe() error = %v, want nil", err)
		}
		subs = append(subs, sub)
	}

	var wg sync.WaitGroup
	for _, sub := range subs {
		wg.Add(1)
		go func(sub Subscription[string]) {
			defer wg.Done()
			_ = sub.Close()
		}(sub)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = stream.Close()
	}()

	wg.Wait()

	for i, sub := range subs {
		select {
		case _, ok := <-sub.Messages():
			if ok {
				t.Fatalf("sub[%d].Messages() open, want closed", i)
			}
		default:
			t.Fatalf("sub[%d].Messages() not closed after concurrent closes", i)
		}
	}
}
