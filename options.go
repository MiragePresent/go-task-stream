package taskstream

import "context"

// StartPosition represents a backend-defined starting point for a subscription.
type StartPosition string

// SubscribeConfig contains parsed subscription option values.
type SubscribeConfig struct {
	Context       context.Context
	StartAt       StartPosition
	ConsumerGroup string
	BufferSize    int
}

// SubscribeOption configures stream subscription behavior.
type SubscribeOption func(*SubscribeConfig) error

// StreamOption configures stream open behavior.
type StreamOption func(*streamConfig) error

type streamConfig struct{}

// WithContext sets the lifecycle context for a subscription.
func WithContext(ctx context.Context) SubscribeOption {
	return func(cfg *SubscribeConfig) error {
		cfg.Context = ctx
		return nil
	}
}

// WithStartAt sets a backend-defined start position for a subscription.
func WithStartAt(startAt StartPosition) SubscribeOption {
	return func(cfg *SubscribeConfig) error {
		cfg.StartAt = startAt
		return nil
	}
}

// WithConsumerGroup sets the consumer-group identifier for competing-consumer
// semantics on supported backends.
func WithConsumerGroup(group string) SubscribeOption {
	return func(cfg *SubscribeConfig) error {
		cfg.ConsumerGroup = group
		return nil
	}
}

// WithBufferSize sets a local subscription channel buffer hint.
func WithBufferSize(size int) SubscribeOption {
	return func(cfg *SubscribeConfig) error {
		cfg.BufferSize = size
		return nil
	}
}
