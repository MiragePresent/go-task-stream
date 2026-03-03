package taskstream

// Publisher publishes messages to a bound stream.
type Publisher[T any] interface {
	// Publish writes msg to the stream and returns a backend message identifier
	// when available.
	Publish(msg Message[T]) (messageID string, err error)
}

// Subscriber creates stream subscriptions.
type Subscriber[T any] interface {
	// Subscribe creates a subscription bound to the stream.
	Subscribe(opts ...SubscribeOption) (Subscription[T], error)
}

// Stream is a composed publish/subscribe abstraction bound to one logical
// stream.
type Stream[T any] interface {
	Publisher[T]
	Subscriber[T]

	// Close releases stream resources. Close is expected to be idempotent by
	// concrete implementations.
	Close() error
}

// Subscription receives messages from a stream and optionally supports
// acknowledgment semantics.
type Subscription[T any] interface {
	// Messages returns the receive-only message channel for this subscription.
	Messages() <-chan Message[T]

	// Ack marks messageID as successfully processed.
	Ack(messageID string) error

	// Nack marks messageID as unsuccessfully processed with the associated
	// reason.
	Nack(messageID string, reason error) error

	// Close releases subscription resources. Close is expected to be idempotent
	// by concrete implementations.
	Close() error
}

// StreamOpener opens a logical stream binding and returns a Stream instance.
type StreamOpener[T any] interface {
	OpenStream(name string, opts ...StreamOption) (Stream[T], error)
}

// OpenStream opens a logical stream by name and returns a bound Stream.
//
// The core package provides lifecycle semantics and option handling for this
// stream. Delivery guarantees and behavior are backend-specific.
func OpenStream[T any](name string, opts ...StreamOption) (Stream[T], error) {
	return openCoreStream[T](name, opts...)
}
