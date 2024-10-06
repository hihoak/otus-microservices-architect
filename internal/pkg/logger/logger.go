package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	jsonHandler := slog.NewJSONHandler(os.Stderr, nil)
	Log = slog.New(jsonHandler)
}
