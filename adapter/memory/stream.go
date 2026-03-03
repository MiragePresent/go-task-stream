package memory

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	taskstream "github.com/miragepresent/go-task-stream"
)

var streamNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type stream[T any] struct {
	mu       sync.RWMutex
	closed   bool
	seq      uint64
	subs     map[*subscription[T]]struct{}
	streamID string
}

// OpenStream opens an in-memory stream bound to name.
//
// In-memory streams support fan-out delivery only. Behavioral subscription
// options such as start position and consumer group return
// taskstream.ErrNotSupported.
func OpenStream[T any](name string, opts ...taskstream.StreamOption) (taskstream.Stream[T], error) {
	if !isValidStreamName(name) {
		return nil, fmt.Errorf("open memory stream %q: %w", name, taskstream.ErrInvalidStream)
	}
	// StreamOption currently cannot be interpreted outside the core package
	// because its config type is package-private. Reject non-nil options
	// explicitly for now.
	for _, opt := range opts {
		if opt != nil {
			return nil, fmt.Errorf("open memory stream option: %w", taskstream.ErrNotSupported)
		}
	}

	return &stream[T]{
		subs:     make(map[*subscription[T]]struct{}),
		streamID: name,
	}, nil
}

func isValidStreamName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	return streamNamePattern.MatchString(name)
}

func (s *stream[T]) Publish(msg taskstream.Message[T]) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return "", taskstream.ErrClosed
	}

	id := msg.ID
	if id == "" {
		id = strconv.FormatUint(atomic.AddUint64(&s.seq, 1), 10)
	}
	msg.ID = id

	for sub := range s.subs {
		sub.deliver(msg.Clone())
	}

	return id, nil
}

func (s *stream[T]) Subscribe(opts ...taskstream.SubscribeOption) (taskstream.Subscription[T], error) {
	cfg, err := parseSubscribeOptions(opts...)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, taskstream.ErrClosed
	}

	sub := &subscription[T]{
		messages: make(chan taskstream.Message[T], cfg.BufferSize),
	}
	sub.onClose = func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.subs, sub)
	}
	s.subs[sub] = struct{}{}

	if cfg.Context != nil && cfg.Context != context.Background() {
		go func() {
			<-cfg.Context.Done()
			_ = sub.Close()
		}()
	}

	return sub, nil
}

func (s *stream[T]) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	subs := make([]*subscription[T], 0, len(s.subs))
	for sub := range s.subs {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Close()
	}

	return nil
}

type subscription[T any] struct {
	mu        sync.Mutex
	messages  chan taskstream.Message[T]
	closed    bool
	closeOnce sync.Once
	onClose   func()
}

func (s *subscription[T]) Messages() <-chan taskstream.Message[T] {
	return s.messages
}

func (s *subscription[T]) Ack(messageID string) error {
	_ = messageID
	return fmt.Errorf("memory adapter ack: %w", taskstream.ErrNotSupported)
}

func (s *subscription[T]) Nack(messageID string, reason error) error {
	_ = messageID
	_ = reason
	return fmt.Errorf("memory adapter nack: %w", taskstream.ErrNotSupported)
}

func (s *subscription[T]) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closed = true
		close(s.messages)
		if s.onClose != nil {
			s.onClose()
		}
	})
	return nil
}

func (s *subscription[T]) deliver(msg taskstream.Message[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.messages <- msg
}

func parseSubscribeOptions(opts ...taskstream.SubscribeOption) (taskstream.SubscribeConfig, error) {
	cfg := taskstream.SubscribeConfig{
		Context: context.Background(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return taskstream.SubscribeConfig{}, err
		}
	}
	if cfg.Context == nil {
		cfg.Context = context.Background()
	}
	if cfg.StartAt != "" {
		return taskstream.SubscribeConfig{}, fmt.Errorf("memory adapter start position: %w", taskstream.ErrNotSupported)
	}
	if cfg.ConsumerGroup != "" {
		return taskstream.SubscribeConfig{}, fmt.Errorf("memory adapter consumer group: %w", taskstream.ErrNotSupported)
	}
	if cfg.BufferSize < 0 {
		cfg.BufferSize = 0
	}
	return cfg, nil
}
