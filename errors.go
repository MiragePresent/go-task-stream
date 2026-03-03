package taskstream

import "errors"

var (
	// ErrClosed indicates an operation was attempted on a closed stream or
	// subscription.
	ErrClosed = errors.New("taskstream: closed")

	// ErrInvalidStream indicates a stream name or stream binding is invalid.
	ErrInvalidStream = errors.New("taskstream: invalid stream")

	// ErrNotSupported indicates the target backend does not support a requested
	// operation or behavior.
	ErrNotSupported = errors.New("taskstream: not supported")
)
