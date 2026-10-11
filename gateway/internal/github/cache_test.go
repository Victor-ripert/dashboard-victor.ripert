package github

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingActivityService struct {
	mu       sync.Mutex
	requests int
	err      error
	started  chan struct{}
	release  chan struct{}
}

func (service *countingActivityService) Activity(_ context.Context, repository string, _ time.Time) (Activity, error) {
	service.mu.Lock()
	service.requests++
	service.mu.Unlock()
	if service.started != nil {
		service.started <- struct{}{}
		<-service.release
	}
	if service.err != nil {
		return Activity{}, service.err
	}
	return Activity{Repository: repository}, nil
}

func (service *countingActivityService) requestCount() int {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.requests
}

func TestCachedClientCachesByRepositoryAndExpiresEntries(t *testing.T) {
	service := &countingActivityService{}
	client := NewCachedClient(service, time.Minute)
	start := time.Date(2026, time.March, 11, 12, 0, 0, 0, time.UTC)

	if _, err := client.Activity(context.Background(), "acme/one", start); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Activity(context.Background(), "acme/one", start.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Activity(context.Background(), "acme/two", start.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Activity(context.Background(), "acme/one", start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if got := service.requestCount(); got != 3 {
		t.Fatalf("source requests = %d, want 3", got)
	}
}

func TestCachedClientDoesNotCacheFailures(t *testing.T) {
	service := &countingActivityService{err: errors.New("GitHub unavailable")}
	client := NewCachedClient(service, time.Minute)
	now := time.Now()

	for range 2 {
		if _, err := client.Activity(context.Background(), "acme/project", now); err == nil {
			t.Fatal("expected source error")
		}
	}
	if got := service.requestCount(); got != 2 {
		t.Fatalf("source requests = %d, want 2", got)
	}
}

func TestCachedClientCoalescesConcurrentRequests(t *testing.T) {
	service := &countingActivityService{started: make(chan struct{}, 1), release: make(chan struct{})}
	client := NewCachedClient(service, time.Minute)
	now := time.Now()
	results := make(chan error, 2)

	go func() {
		_, err := client.Activity(context.Background(), "acme/project", now)
		results <- err
	}()
	<-service.started
	go func() {
		_, err := client.Activity(context.Background(), "acme/project", now)
		results <- err
	}()
	close(service.release)

	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := service.requestCount(); got != 1 {
		t.Fatalf("source requests = %d, want 1", got)
	}
}

func TestCachedClientSupportsManyConcurrentCallers(t *testing.T) {
	service := &countingActivityService{}
	client := NewCachedClient(service, time.Minute)
	now := time.Now()
	var failures atomic.Int32
	var waitGroup sync.WaitGroup
	for range 20 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if _, err := client.Activity(context.Background(), "acme/project", now); err != nil {
				failures.Add(1)
			}
		}()
	}
	waitGroup.Wait()
	if failures.Load() != 0 || service.requestCount() != 1 {
		t.Fatalf("failures = %d, source requests = %d; want 0 failures and 1 request", failures.Load(), service.requestCount())
	}
}
