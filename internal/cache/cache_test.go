package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"gh-pin-repo-go/internal/model"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(1 * time.Minute)
	defer c.Close()

	testData := []model.PinnedRepo{
		{Owner: "test", Repo: "repo1", Stars: 100},
	}

	c.Set("user1", testData, 1*time.Minute)

	res, found := c.Get("user1")
	if !found {
		t.Fatalf("expected key 'user1' to be found")
	}
	if len(res) != 1 || res[0].Repo != "repo1" {
		t.Fatalf("unexpected cached data: %+v", res)
	}

	// Non-existent key
	_, found = c.Get("unknown")
	if found {
		t.Fatalf("expected 'unknown' not to be found")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(50 * time.Millisecond)
	defer c.Close()

	testData := []model.PinnedRepo{{Repo: "temp"}}
	c.Set("temp_user", testData, 50*time.Millisecond)

	// Immediate get
	_, found := c.Get("temp_user")
	if !found {
		t.Fatalf("expected key to exist initially")
	}

	// Wait for expiration
	time.Sleep(70 * time.Millisecond)
	_, found = c.Get("temp_user")
	if found {
		t.Fatalf("expected key to be expired and not found")
	}
}

func TestCache_HighConcurrency(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Close()

	var wg sync.WaitGroup
	workers := 50
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			key := fmt.Sprintf("user_%d", workerID%10)
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					c.Set(key, []model.PinnedRepo{{Repo: fmt.Sprintf("repo_%d", j)}}, 5*time.Minute)
				} else {
					c.Get(key)
				}
			}
		}(i)
	}

	wg.Wait()
}
