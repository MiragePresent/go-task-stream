// Package memory provides an optional in-memory stream adapter for local
// development and tests.
//
// Behavior:
//   - non-persistent, process-local transport
//   - fan-out delivery to direct subscribers
//   - best-effort delivery; messages are not replayed after process restart
//   - ordering is best-effort per publish call; no global strict ordering guarantee
//   - no offset or durable cursor model
//   - no retry/redelivery mechanism
//   - no consumer group support
//   - Ack/Nack return taskstream.ErrNotSupported
//   - unsupported behavioral options: WithStartAt, WithConsumerGroup
package memory
