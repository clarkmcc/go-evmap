package eventual

// OptionFunc allows customizing the Options with functions
type OptionFunc func(*Options)

type Options struct {
	// MaxReplicationWriteLag determines the maximum number of writes that the map can
	// observe before those writes are replicated to the readers.
	MaxReplicationWriteLag int
}

// WithMaxReplicationWriteLag sets the MaxReplicationWriteLag
func WithMaxReplicationWriteLag(writes int) OptionFunc {
	return func(options *Options) {
		options.MaxReplicationWriteLag = writes
	}
}
