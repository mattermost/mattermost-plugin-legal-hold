package jobs

type loggerIface interface {
	// Error logs an error message, optionally structured with alternating key, value parameters.
	Error(message string, keyValuePairs ...any)

	// Warn logs an error message, optionally structured with alternating key, value parameters.
	Warn(message string, keyValuePairs ...any)

	// Info logs an error message, optionally structured with alternating key, value parameters.
	Info(message string, keyValuePairs ...any)

	// Debug logs an error message, optionally structured with alternating key, value parameters.
	Debug(message string, keyValuePairs ...any)
}
