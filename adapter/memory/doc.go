// Package memory provides an optional in-memory stream adapter for local
// development and tests.
//
// Behavior:
//   - non-persistent, process-local transport
//   - fan-out delivery to direct subscribers
//   - no consumer group support
//   - Ack/Nack return taskstream.ErrNotSupported
package memory
