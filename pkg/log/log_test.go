package log

import (
	"context"
	"testing"
)

func TestLogger(t *testing.T) {
	l := NewLogger("")
	l.Info("msg1", "k1", "v1")

	ctx := context.Background()
	ctx = AppendCtx(ctx, "trace", "id1")
	l.InfoC(ctx, "msg2", "k2", "v2")

	l.InfoF("msg3: %s=%s", "k3", "v3")
	ctx = AppendCtx(ctx, "trace", "id2")
	l.InfoFC(ctx, "msg4: %s=%s", "k4", "v4")
}
