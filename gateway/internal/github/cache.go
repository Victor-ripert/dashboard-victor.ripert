package github

import (
	"context"
	"sync"
	"time"
)

const activityCacheTTL = time.Minute

type CachedClient struct {
	source   ActivityService
	ttl      time.Duration
	mu       sync.Mutex
	entries  map[string]cacheEntry
	inFlight map[string]*cacheCall
}

type cacheEntry struct {
	activity Activity
	expires  time.Time
}

type cacheCall struct {
	done     chan struct{}
	activity Activity
	err      error
}

func NewCachedClient(source ActivityService, ttl time.Duration) *CachedClient {
	if ttl <= 0 {
		ttl = activityCacheTTL
	}
	return &CachedClient{
		source:   source,
		ttl:      ttl,
		entries:  make(map[string]cacheEntry),
		inFlight: make(map[string]*cacheCall),
	}
}

func (client *CachedClient) Activity(ctx context.Context, repository string, now time.Time) (Activity, error) {
	client.mu.Lock()
	if entry, ok := client.entries[repository]; ok && now.Before(entry.expires) {
		client.mu.Unlock()
		return entry.activity, nil
	}
	if call, ok := client.inFlight[repository]; ok {
		client.mu.Unlock()
		select {
		case <-call.done:
			return call.activity, call.err
		case <-ctx.Done():
			return Activity{}, ctx.Err()
		}
	}
	call := &cacheCall{done: make(chan struct{})}
	client.inFlight[repository] = call
	client.mu.Unlock()

	activity, err := client.source.Activity(ctx, repository, now)

	client.mu.Lock()
	call.activity = activity
	call.err = err
	if err == nil {
		client.entries[repository] = cacheEntry{activity: activity, expires: now.Add(client.ttl)}
	}
	delete(client.inFlight, repository)
	close(call.done)
	client.mu.Unlock()
	return activity, err
}
