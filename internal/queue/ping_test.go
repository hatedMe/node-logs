package queue

import (
	"context"
	"testing"
	"time"
)

func TestProducerPing(t *testing.T) {
	p, cleanup := testProducer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Ping(ctx); err != nil {
		t.Fatalf("Ping 应成功: %v", err)
	}
}
