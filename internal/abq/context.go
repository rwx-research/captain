package abq

import (
	"context"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
}

var stateFilePathKey = contextKey("abq-state-file-path")

func WithStateFilePath(ctx context.Context, stateFile string) context.Context {
	return context.WithValue(ctx, stateFilePathKey, stateFile)
}

func StateFilePath(ctx context.Context) string {
	if value, ok := ctx.Value(stateFilePathKey).(string); ok {
		return value
	}
	return ""
}
