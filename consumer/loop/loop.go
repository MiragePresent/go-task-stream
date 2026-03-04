package loop

import (
	"context"
	"errors"
	"fmt"

	taskstream "github.com/miragepresent/go-task-stream"
)

// Handler processes one message from a stream subscription.
type Handler[T any] func(ctx context.Context, msg taskstream.Message[T]) error

// ErrorHandler receives per-message processing errors from Consume while the
// loop continues processing subsequent messages.
type ErrorHandler[T any] func(ctx context.Context, msg taskstream.Message[T], err error)

// Consume reads messages from sub and applies handler for each message until
// ctx is canceled or the subscription channel closes.
//
// On handler success, Consume calls Ack. On handler failure, Consume calls
// Nack and then continues processing. Ack/Nack ErrNotSupported is ignored to
// support backends without acknowledgment features.
//
// Non-acknowledgment handler errors can be surfaced through ConsumeWithErrors.
func Consume[T any](ctx context.Context, sub taskstream.Subscription[T], handler Handler[T]) error {
	return ConsumeWithErrors(ctx, sub, handler, nil)
}

// ConsumeWithErrors behaves like Consume but reports per-message handler
// failures to onError before continuing with the next message.
func ConsumeWithErrors[T any](
	ctx context.Context,
	sub taskstream.Subscription[T],
	handler Handler[T],
	onError ErrorHandler[T],
) error {
	if handler == nil {
		return errors.New("loop consume: nil handler")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-sub.Messages():
			if !ok {
				return nil
			}
			if err := handler(ctx, msg); err != nil {
				if nackErr := sub.Nack(msg.ID, err); nackErr != nil && !errors.Is(nackErr, taskstream.ErrNotSupported) {
					return fmt.Errorf("loop consume nack: %w", nackErr)
				}
				if onError != nil {
					onError(ctx, msg, err)
				}
				continue
			}
			if ackErr := sub.Ack(msg.ID); ackErr != nil && !errors.Is(ackErr, taskstream.ErrNotSupported) {
				return fmt.Errorf("loop consume ack: %w", ackErr)
			}
		}
	}
}
