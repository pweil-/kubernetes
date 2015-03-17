package types

// Labels allows you to present labels independently from their storage.
type Labels interface {
	// Has returns whether the provided label exists.
	Has(label string) (exists bool)

	// Get returns the value for the provided label.
	Get(label string) (value string)

	String() string
	AsSelector() Selector
}

// Set is a map of label:value.
type Set map[string]string

// Selector represents a label selector.
type Selector interface {
	// Matches returns true if this selector matches the given set of labels.
	Matches(Labels) bool

	// Empty returns true if this selector does not restrict the selection space.
	Empty() bool

	// RequiresExactMatch allows a caller to introspect whether a given selector
	// requires a single specific label to be set, and if so returns the value it
	// requires.
	RequiresExactMatch(label string) (value string, found bool)

	// String returns a human readable string that represents this selector.
	String() string
}
