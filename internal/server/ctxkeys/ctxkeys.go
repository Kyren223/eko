// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

func (k key) String() string {
	return keyNames[k]
}

func init() {
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
