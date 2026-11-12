package hidtransport

// RetryTransient exposes the write retry rule to the external tests.
func RetryTransient(call func() error) error { return retryTransient(call) }

// WriteAttempts is how many times a write is tried.
const WriteAttempts = writeAttempts
