package accounting_bot

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type NamespaceHook struct {
	ns string
}

func (h *NamespaceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *NamespaceHook) Fire(e *logrus.Entry) error {
	b := strings.Builder{}
	b.Grow(len(e.Message) + len(h.ns) + 3)
	b.WriteString("(")
	b.WriteString(h.ns)
	b.WriteString(") ")
	b.WriteString(e.Message)
	e.Message = b.String()
	return nil
}

func NewNamespaceHook(ns string) *NamespaceHook {
	return &NamespaceHook{ns: ns}
}

func NewLogger(level logrus.Level, namespace string) *logrus.Logger {
	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{DisableSorting: false},
		Hooks:     make(logrus.LevelHooks),
		Level:     level,
		ExitFunc:  os.Exit,
	}
	if len(namespace) > 0 {
		logger.AddHook(NewNamespaceHook(namespace))
	}
	return logger
}
