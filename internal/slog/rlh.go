package slog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisHandler struct {
	client     *redis.Client
	level      slog.Level
	attrs      []slog.Attr
	streamName string
	handle     func(h *RedisHandler, ctx context.Context, r slog.Record) error
}

type LogJson struct {
	Time    time.Time
	Level   slog.Level
	Message string
}

func NewStreamHandler(rClient *redis.Client, level slog.Level) *RedisHandler {
	return new(rClient, level, handleStream)
}
func NewJsonHandler(rClient *redis.Client, level slog.Level) *RedisHandler {
	return new(rClient, level, handleJson)
}
func NewPubSubHandler(rClient *redis.Client, level slog.Level) *RedisHandler {
	return new(rClient, level, handlePubSub)
}

func new(rClient *redis.Client, level slog.Level, handle func(h *RedisHandler, ctx context.Context, r slog.Record) error) *RedisHandler {
	return &RedisHandler{
		client:     rClient,
		level:      level,
		streamName: "rlh-stream",
		handle:     handle,
	}
}

func (h *RedisHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handle(h, ctx, r)
}
func (h *RedisHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}
func (h *RedisHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}
func (h *RedisHandler) WithGroup(name string) slog.Handler {
	clone := *h
	return &clone
}

func getLogMap(r slog.Record) map[string]any {
	return map[string]any{
		"time":    r.Time.Format(time.RFC3339),
		"level":   r.Level.String(),
		"message": r.Message,
		"extra":   "",
	}
}
func handleStream(h *RedisHandler, ctx context.Context, r slog.Record) error {
	logMap := getLogMap(r)
	extraMap := make(map[string]any)

	r.Attrs(func(a slog.Attr) bool {
		extraMap[a.Key] = a.Value.String()
		return true
	})

	if len(extraMap) > 0 {
		data, err := json.Marshal(extraMap)
		if err != nil {
			return err
		}
		logMap["extra"] = data
	}

	err := h.client.XAdd(ctx, &redis.XAddArgs{
		Stream: h.streamName,
		ID:     "*",
		Values: logMap,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func handleJson(h *RedisHandler, ctx context.Context, r slog.Record) error {
	logMap := getLogMap(r)
	key := ""

	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "x-rlh-key" {
			key = fmt.Sprintf("%s_%s", r.Level, a.Value.String())
			return true
		}

		logMap[a.Key] = a.Value.String()
		return true
	})

	if key == "" {
		key = uuid.New().String()
	}

	data, err := json.Marshal(logMap)
	if err != nil {
		return err
	}

	err = h.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func handlePubSub(h *RedisHandler, ctx context.Context, r slog.Record) error {
	return nil
}
