package jobs

type loggerIface interface {
	// Error logs an error message, optionally structured with alternating key, value parameters.
	Error(message string, keyValuePairs ...interface{})

	// Warn logs an error message, optionally structured with alternating key, value parameters.
	Warn(message string, keyValuePairs ...interface{})

	// Info logs an error message, optionally structured with alternating key, value parameters.
	Info(message string, keyValuePairs ...interface{})

	// Debug logs an error message, optionally structured with alternating key, value parameters.
	Debug(message string, keyValuePairs ...interface{})
}
