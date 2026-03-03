// Package taskstream defines the backend-agnostic core API for stream-based task
// processing.
//
// Scope:
//   - Typed message envelope and cloning
//   - Stream publish/subscribe contracts
//   - Subscription ack/nack contracts
//   - Stream/subscription option types
//   - Sentinel errors for common capability and lifecycle states
//
// Guarantees:
//   - API-level contracts are transport-neutral
//   - Capability mismatches are represented with ErrNotSupported
//   - Tuning options may fall back to backend defaults when unsupported
//     directly; adapter docs define exact fallback behavior
//
// Non-goals in core v1:
//   - Exactly-once semantics
//   - Workflow DSL/orchestration
//   - Built-in distributed tracing exporters
//   - Full Kafka/Rabbit adapter implementations
package taskstream
