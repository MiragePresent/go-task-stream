package taskstream

// Message is the transport-neutral task envelope used by the core API.
type Message[T any] struct {
	// ID is an optional backend message identifier.
	ID string

	// Type identifies the message kind.
	Type string

	// Data contains the typed payload.
	Data T

	// Meta contains transport/runtime metadata such as timestamps and retry
	// attempt info.
	Meta map[string]any
}

// Clone returns a copy of m.
//
// Map values in Meta are copied by assignment. If a value is itself a
// reference type (for example a map or slice), that nested value remains
// shared.
func (m Message[T]) Clone() Message[T] {
	cloned := m

	if m.Meta != nil {
		cloned.Meta = make(map[string]any, len(m.Meta))
		for k, v := range m.Meta {
			cloned.Meta[k] = v
		}
	}

	return cloned
}
