package video_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/video"
)

func TestMemoryCache_SetGetWithinTTL(t *testing.T) {
	ctx := context.Background()
	c := video.NewMemoryCache()
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, found, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v" {
		t.Errorf("Get = (%q, %v), want (v, true)", val, found)
	}
}

func TestMemoryCache_Expiry(t *testing.T) {
	ctx := context.Background()
	c := video.NewMemoryCache()
	if err := c.Set(ctx, "k", []byte("v"), 10*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	_, found, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected entry to be expired")
	}
}

func TestMemoryCache_MissNoPanic(t *testing.T) {
	ctx := context.Background()
	c := video.NewMemoryCache()
	_, found, err := c.Get(ctx, "absent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected found=false on a missing key")
	}
}

func TestMemoryCache_NoExpiryWhenTTLZero(t *testing.T) {
	ctx := context.Background()
	c := video.NewMemoryCache()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	_, found, _ := c.Get(ctx, "k")
	if !found {
		t.Error("ttl<=0 should mean no expiry")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	c := video.NewMemoryCache()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Set(ctx, "k", []byte("v"), time.Minute)
			_, _, _ = c.Get(ctx, "k")
		}()
	}
	wg.Wait() // -race must report no data race
}

// MemoryCache must satisfy the Cache interface.
var _ video.Cache = (*video.MemoryCache)(nil)
