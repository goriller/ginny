package logger

type Level int8

const (
	// debugLevel level. Usually only enabled when debugging. Very verbose logging.
	debugLevel Level = iota
	// infoLevel is the default logging priority.
	// General operational entries about what's going on inside the application.
	infoLevel
	// warnLevel level. Non-critical entries that deserve eyes.
	warnLevel
	// errorLevel level. Logs. Used for errors that should definitely be noted.
	errorLevel
	// fatalLevel level. Logs and then calls `logger.Exit(1)`. highest level of severity.
	fatalLevel
)
