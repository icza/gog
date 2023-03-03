package gog

import (
	"context"
	"sync"
	"time"
)

// RunEvictor should be run as a goroutine, it evicts expired cache entries from the listed OpCaches.
// Returns only if ctx is cancelled.
//
// OpCache has Evict() method, so any OpCache can be listed (does not depend on the type parameter).
func RunEvictor(ctx context.Context, evictorPeriod time.Duration, opCaches ...interface{ Evict() }) {
	ticker := time.NewTicker(evictorPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		for _, oc := range opCaches {
			oc.Evict()
		}
	}
}

// OpCacheConfig holds configuration options for OpCache.
type OpCacheConfig struct {
	// Operation results are valid for this long after creation.
	ResultExpiration time.Duration

	// Expired results are still usable for this long after expiration.
	// Tip: if this field is 0, grace period and thus background
	// op execution will be disabled.
	ResultGraceExpiration time.Duration
}

// OpCache implements a general value cache.
// It can be used to cache results of arbitrary operations.
//
// Operations are captured by a function that returns a value of a certain type (T) and an error.
// If an operation has multiple results beside the error, they must be wrapped in a struct or slice.
// Operations are identified by a string key.
type OpCache[T any] struct {
	cfg OpCacheConfig

	keyResultsMu sync.RWMutex
	keyResults   map[string]*opResult[T]
}

// NewOpCache creates a new OpCache.
func NewOpCache[T any](cfg OpCacheConfig) *OpCache[T] {
	return &OpCache[T]{
		cfg:        cfg,
		keyResults: map[string]*opResult[T]{},
	}
}

func (oc *OpCache[T]) getCachedOpResult(key string) *opResult[T] {
	oc.keyResultsMu.RLock()
	defer oc.keyResultsMu.RUnlock()

	return oc.keyResults[key]
}

func (oc *OpCache[T]) setCachedOpResult(key string, opResults *opResult[T]) {
	oc.keyResultsMu.Lock()
	oc.keyResults[key] = opResults
	oc.keyResultsMu.Unlock()
}

// Evict checks all cached entries, and removes invalid ones.
func (oc *OpCache[T]) Evict() {
	expiration := oc.cfg.ResultExpiration + oc.cfg.ResultGraceExpiration

	oc.keyResultsMu.Lock()
	defer oc.keyResultsMu.Unlock()

	for key, opResult := range oc.keyResults {
		if !opResult.valid(expiration) {
			delete(oc.keyResults, key)
		}
	}
}

// Get gets the result of an operation.
//
// If the result is cached and valid, it is returned immediately.
//
// If the result is cached but not valid, but we're within the grace period,
// execOp() is called in the background to refresh the cache,
// and the cached result is returned immediately.
// Care is taken to only launch a single background worker to refresh the cache even if
// Get() is called multiple times with the same key before the cache can be refreshed.
//
// Else result is either not cached or we're past the grace period:
// execOp() is executed, the function waits for its return values, the result is cached,
// and then the fresh result is returned.
func (oc *OpCache[T]) Get(
	key string,
	execOp func() (result T, err error),
) (result T, resultErr error) {

	cachedResult := oc.getCachedOpResult(key)

	if cachedResult.valid(oc.cfg.ResultExpiration) {
		return cachedResult.result, cachedResult.resultErr
	}

	if oc.cfg.ResultGraceExpiration <= 0 || !cachedResult.valid(oc.cfg.ResultExpiration+oc.cfg.ResultGraceExpiration) {
		// Not valid and not even within grace period: query and cache unconditionally:
		result, err := execOp()
		oc.setCachedOpResult(key, newOpResult(result, err))
		return result, err
	}

	// Cached result is within grace period, we can use it:
	result, resultErr = cachedResult.result, cachedResult.resultErr

	// But need to reload, in the background.
	// First use read-lock to check if someone's already doing it:

	cachedResult.reloadMu.RLock()
	reloading := cachedResult.reloading
	cachedResult.reloadMu.RUnlock()
	if reloading {
		// Already reloading, nothing to do
		return
	}

	// Try to take ownership of reloading, needs write-lock:
	cachedResult.reloadMu.Lock()
	if cachedResult.reloading {
		// Someone else got the write-lock first, he'll take care of the reload
		cachedResult.reloadMu.Unlock()
		return
	}
	cachedResult.reloading = true // We'll be the one doing it
	cachedResult.reloadMu.Unlock()

	// reload in new goroutine
	// Note: must use function literal, else the function param (execOp()) would be evaluated (called) in this goroutine!
	go func() {
		oc.setCachedOpResult(key, newOpResult(execOp()))
	}()

	return
}

// opResult holds the result of an operation.
type opResult[T any] struct {
	created time.Time

	result    T // If an op has multiple results, this should be a slice (e.g. []any)
	resultErr error

	reloadMu  sync.RWMutex
	reloading bool
}

// newOpResult creates a new OpResult.
func newOpResult[T any](result T, resultErr error) *opResult[T] {
	return &opResult[T]{
		created:   time.Now(),
		result:    result,
		resultErr: resultErr,
	}
}

// valid tells if the result is valid.
func (opr *opResult[T]) valid(expiration time.Duration) bool {
	return opr != nil && time.Since(opr.created) < expiration
}
