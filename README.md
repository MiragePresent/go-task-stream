# go-task-stream

`go-task-stream` is a backend-agnostic Go package for stream-driven task processing.

It defines typed core contracts for:
- Opening a logical stream.
- Publishing typed messages.
- Subscribing to messages.
- Handling lifecycle and capability errors consistently.

## Scope and Architecture

Core package responsibilities:
- `Message[T]` envelope with clone support.
- Stream contracts: `Stream[T]`, `Publisher[T]`, `Subscriber[T]`, `Subscription[T]`.
- Stream binding API: `OpenStream[T](name string, opts ...StreamOption)`.
- Subscribe option model and sentinel errors.

Adapter responsibilities (outside core):
- Transport implementation (in-memory, Redis, Kafka, RabbitMQ, etc.).
- Ordering guarantees.
- Persistence and redelivery behavior.
- Ack/nack semantics and retry behavior.

## Backend Capability Model

Capabilities vary by backend. Callers should feature-detect via errors:
- `ErrNotSupported`: requested behavior is not implemented by the backend.
- `ErrClosed`: operation attempted on a closed stream/subscription.
- `ErrInvalidStream`: invalid stream name/binding request.

Examples:
- `WithStartAt(...)` and `WithConsumerGroup(...)` may return `ErrNotSupported`.
- `Ack(...)`/`Nack(...)` may return `ErrNotSupported` for backends without ack support.

## Terminology

Public API term is **Message**.

Backend-native terms should stay adapter-specific:
- In-memory channels: item/value.
- Redis Streams: entry.
- Kafka: record.
- RabbitMQ: delivery.

## Stream Binding and Lifecycle

Intended usage flow:
1. Open a typed stream with a logical name.
2. Create subscriptions and run consumers.
3. Publish typed messages.
4. Close subscriptions/stream during shutdown (idempotent close).

Notes:
- After `Close()`, `Publish(...)` and `Subscribe(...)` return `ErrClosed`.
- Closing a stream closes active subscription channels safely.

## Quick Start

```go
package main

import (
	"errors"

	taskstream "github.com/miragepresent/go-task-stream"
)

type Action struct {
	Name string
}

func main() {
	stream, _ := taskstream.OpenStream[Action]("actions")
	defer stream.Close()

	sub, _ := stream.Subscribe(taskstream.WithBufferSize(64))
	defer sub.Close()

	_, err := stream.Publish(taskstream.Message[Action]{
		ID:   "1",
		Type: "action",
		Data: Action{Name: "run"},
	})
	if errors.Is(err, taskstream.ErrNotSupported) {
		// Use an adapter with publish delivery support.
	}
}
```

## Current Status

- Core contract, lifecycle semantics, and race-tested concurrency behavior are implemented.
- Optional helper/runtime packages are available:
  - `consumer/loop`: reusable message loop consumer.
  - `consumer/pipeline`: ordered step pipeline consumer with optional output routing.
- Optional adapter package is available:
  - `adapter/memory`: in-memory fan-out stream adapter for local development/testing.
