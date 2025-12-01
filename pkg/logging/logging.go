package logging

import (
    "os"
    "strings"
    "github.com/sirupsen/logrus"
)

func Setup(level string, format string) {
    if level == "" {
        level = strings.ToLower(os.Getenv("KTIB_LOG_LEVEL"))
    }
    if format == "" {
        format = strings.ToLower(os.Getenv("KTIB_LOG_FORMAT"))
    }
    switch level {
    case "trace":
        logrus.SetLevel(logrus.TraceLevel)
    case "debug":
        logrus.SetLevel(logrus.DebugLevel)
    case "warn":
        logrus.SetLevel(logrus.WarnLevel)
    case "error":
        logrus.SetLevel(logrus.ErrorLevel)
    case "fatal":
        logrus.SetLevel(logrus.FatalLevel)
    case "panic":
        logrus.SetLevel(logrus.PanicLevel)
    default:
        logrus.SetLevel(logrus.InfoLevel)
    }
    switch format {
    case "json":
        logrus.SetFormatter(&logrus.JSONFormatter{})
    default:
        logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
    }
    logrus.SetOutput(os.Stdout)
}
