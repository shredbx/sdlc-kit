package rss_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/rss"
)

func TestMemoryCache_SetGet(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, found, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if string(val) != "v" {
		t.Errorf("val = %q, want v", val)
	}
}

func TestMemoryCache_Miss(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()
	_, found, err := c.Get(ctx, "absent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected found=false on miss")
	}
}

func TestMemoryCache_Expiry(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()
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

func TestMemoryCache_NoExpiryWhenTTLZero(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	_, found, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Error("ttl<=0 should mean no expiry")
	}
}

func TestMemoryCache_Overwrite(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()
	_ = c.Set(ctx, "k", []byte("one"), time.Minute)
	_ = c.Set(ctx, "k", []byte("two"), time.Minute)
	val, _, _ := c.Get(ctx, "k")
	if string(val) != "two" {
		t.Errorf("val = %q, want two", val)
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	c := rss.NewMemoryCache()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "k"
			_ = c.Set(ctx, key, []byte("v"), time.Minute)
			_, _, _ = c.Get(ctx, key)
		}(i)
	}
	wg.Wait() // -race must report no data race
}

// MemoryCache must satisfy the Cache interface.
var _ rss.Cache = (*rss.MemoryCache)(nil)
