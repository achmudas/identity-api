package logger

import "log"

type MigrateLogger struct{}

func (l *MigrateLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *MigrateLogger) Verbose() bool {
	return false
}
