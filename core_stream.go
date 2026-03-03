package taskstream

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var streamNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type coreStream[T any] struct {
	mu     sync.RWMutex
	closed bool
	subs   map[*coreSubscription[T]]struct{}
}

func openCoreStream[T any](name string, opts ...StreamOption) (Stream[T], error) {
	if !isValidStreamName(name) {
		return nil, fmt.Errorf("open stream %q: %w", name, ErrInvalidStream)
	}
	if _, err := parseStreamOptions(opts...); err != nil {
		return nil, err
	}

	return &coreStream[T]{
		subs: make(map[*coreSubscription[T]]struct{}),
	}, nil
}

func isValidStreamName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	return streamNamePattern.MatchString(name)
}

func (s *coreStream[T]) Publish(msg Message[T]) (string, error) {
	_ = msg

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return "", ErrClosed
	}
	return "", fmt.Errorf("publish: %w", ErrNotSupported)
}

func (s *coreStream[T]) Subscribe(opts ...SubscribeOption) (Subscription[T], error) {
	cfg, err := parseSubscribeOptions(opts...)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrClosed
	}

	sub := &coreSubscription[T]{
		messages: make(chan Message[T], cfg.BufferSize),
		config:   cfg,
	}
	sub.onClose = func() {
		s.removeSub(sub)
	}
	s.subs[sub] = struct{}{}

	if ctx := cfg.Context; ctx != nil && ctx != context.Background() {
		go func() {
			<-ctx.Done()
			_ = sub.Close()
		}()
	}

	return sub, nil
}

func (s *coreStream[T]) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	subs := make([]*coreSubscription[T], 0, len(s.subs))
	for sub := range s.subs {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Close()
	}

	return nil
}

func (s *coreStream[T]) removeSub(sub *coreSubscription[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, sub)
}

type coreSubscription[T any] struct {
	messages  chan Message[T]
	config    SubscribeConfig
	closeOnce sync.Once
	onClose   func()
}

func (s *coreSubscription[T]) Messages() <-chan Message[T] {
	return s.messages
}

func (s *coreSubscription[T]) Ack(messageID string) error {
	_ = messageID
	return fmt.Errorf("ack: %w", ErrNotSupported)
}

func (s *coreSubscription[T]) Nack(messageID string, reason error) error {
	_ = messageID
	_ = reason
	return fmt.Errorf("nack: %w", ErrNotSupported)
}

func (s *coreSubscription[T]) Close() error {
	s.closeOnce.Do(func() {
		close(s.messages)
		if s.onClose != nil {
			s.onClose()
		}
	})
	return nil
}
