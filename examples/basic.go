package main

import (
	"context"
	"log/slog"
	"time"

	rlh "github.com/paragonov/go-redis-log-handler/internal/slog"
	"github.com/redis/go-redis/v9"
)

func main() {
	opt, err := redis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opt)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	rlh := rlh.NewJsonHandler(rdb, slog.LevelDebug)
	logger := slog.New(rlh)

	passArgs := make([]any, 0, 6)
	passArgs = append(passArgs,
		"x-request-id", 1,
		"x-rlh-key", "key",
	)

	logger.Info("application started", passArgs...)
	logger.Debug("this will not be shown at info level", passArgs...)
	logger.Warn("something suspicious happened", passArgs...)
	logger.Error("something failed", passArgs...)
}
