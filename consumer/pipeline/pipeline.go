package pipeline

import (
	"context"
	"errors"
	"fmt"

	taskstream "github.com/miragepresent/go-task-stream"
)

// Step transforms msg in-order as part of a processing pipeline.
type Step[T any] func(ctx context.Context, msg taskstream.Message[T]) (taskstream.Message[T], error)

// Router handles optional pipeline output routing.
type Router[T any] func(ctx context.Context, msg taskstream.Message[T]) error

// ErrorHandler receives per-message pipeline failures while Consume continues
// processing subsequent messages.
type ErrorHandler[T any] func(ctx context.Context, msg taskstream.Message[T], err error)

// Consume reads messages from sub and applies ordered steps for each message.
//
// Pipeline behavior:
//   - stops the current message on first step error
//   - runs router after all steps when router is configured
//   - calls Nack on step/router failure when supported
//   - calls Ack only after full success when supported
//   - continues with next message after step/router failure
//
// Ack/Nack ErrNotSupported is ignored to support backends that do not expose
// explicit acknowledgment APIs.
func Consume[T any](
	ctx context.Context,
	sub taskstream.Subscription[T],
	steps []Step[T],
	router Router[T],
) error {
	return ConsumeWithErrors(ctx, sub, steps, router, nil)
}

// ConsumeWithErrors behaves like Consume but reports per-message step/router
// failures to onError before continuing with the next message.
func ConsumeWithErrors[T any](
	ctx context.Context,
	sub taskstream.Subscription[T],
	steps []Step[T],
	router Router[T],
	onError ErrorHandler[T],
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for i, step := range steps {
		if step == nil {
			return fmt.Errorf("pipeline consume: nil step at index %d", i)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-sub.Messages():
			if !ok {
				return nil
			}

			current := msg
			var runErr error
			for _, step := range steps {
				current, runErr = step(ctx, current)
				if runErr != nil {
					break
				}
			}
			if runErr == nil && router != nil {
				runErr = router(ctx, current)
			}

			if runErr != nil {
				if nackErr := sub.Nack(msg.ID, runErr); nackErr != nil && !errors.Is(nackErr, taskstream.ErrNotSupported) {
					return fmt.Errorf("pipeline consume nack: %w", nackErr)
				}
				if onError != nil {
					onError(ctx, msg, runErr)
				}
				continue
			}

			if ackErr := sub.Ack(msg.ID); ackErr != nil && !errors.Is(ackErr, taskstream.ErrNotSupported) {
				return fmt.Errorf("pipeline consume ack: %w", ackErr)
			}
		}
	}
}
