package ctxkeys

import (
	"context"
	"log/slog"

	"github.com/kyren223/eko/pkg/assert"
)

type key int

const (
	UserID key = iota
	IpAddr
	KeyMax
)

var keyNames = map[key]string{
	UserID: "user_id",
	IpAddr: "ip_addr",
}

func Init() {
	assert.Assert(len(keyNames) == int(KeyMax), "Keys in keyNames mismatch amount of keys", "len(keyNames)", len(keyNames), "KeyMax", int(KeyMax))
}

func WithValue(ctx context.Context, k key, v any) context.Context {
	return context.WithValue(ctx, k, v)
}

func Value(ctx context.Context, k key) any {
	return ctx.Value(k)
}

type ContextHandler struct {
	slog.Handler
}

func WrapLogHandler(baseHandler slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: baseHandler}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	for k := key(0); k < KeyMax; k++ {
		if v := Value(ctx, k); v != nil {
			r.AddAttrs(slog.Any(keyNames[k], v))
		}
	}

	return h.Handler.Handle(ctx, r)
}
