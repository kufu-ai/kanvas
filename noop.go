package kanvas

// Noop is a noop configuration that does nothing
// This is mainly for template components that are only used as dependencies.
// You override or replaces this with a real component in the environment.
type Noop struct {
}
