package gog

import (
	"context"
	"crypto/sha1"
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

	// ErrorExpiration is an optional function.
	// If provided, it will be called for non-nil operation errors.
	// If it returns discard=true, the error result will not be cached.
	// Non-nil returned expirations override the default ResultExpiration and ResultGraceExpiration.
	//
	// If provided, this function is only called once for the result error of a single operation execution
	// (regardless of how many times it is accessed from the OpCache).
	ErrorExpiration func(err error) (discard bool, expiration, graceExpiration *time.Duration)
}

// OpCache implements a general value cache.
// It can be used to cache results of arbitrary operations.
// Cached values are tied to a string key that should be derived from the operation's parameters.
// Cached values have an expiration time and also a grace period during which the cached value
// is considered valid, but getting a cached value during the grace period triggers a reload
// that will happen in the background (the cached value is returned immediately, without waiting).
//
// Operations are captured by a function that returns a value of a certain type (T) and an error.
// If an operation has multiple results beside the error, they must be wrapped in a struct or slice.
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
	oc.keyResultsMu.Lock()
	defer oc.keyResultsMu.Unlock()

	for key, opResult := range oc.keyResults {
		if !opResult.graceValid() { // Delete if not even grace-valid
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
// Else result is either not cached or we're past even the grace period:
// execOp() is executed, the function waits for its return values, the result is cached,
// and then the fresh result is returned.
func (oc *OpCache[T]) Get(
	key string,
	execOp func() (result T, err error),
) (result T, resultErr error) {
	key = transformKey(key)

	cachedResult := oc.getCachedOpResult(key)

	if cachedResult.valid() {
		return cachedResult.result, cachedResult.resultErr
	}

	// This function executes execOp(), caches the result according to the configuration, and returns it
	execOpAndCache := func() (result T, resultErr error) {
		result, resultErr = execOp()
		expiration, graceExpiration := oc.cfg.ResultExpiration, oc.cfg.ResultGraceExpiration
		if resultErr != nil && oc.cfg.ErrorExpiration != nil {
			discard, exp, graceExp := oc.cfg.ErrorExpiration(resultErr)
			if discard {
				// This error result is not to be cached at all, just return:
				return
			}
			if exp != nil {
				expiration = *exp
			}
			if graceExp != nil {
				graceExpiration = *graceExp
			}
		}
		oc.setCachedOpResult(key, newOpResult(result, resultErr, expiration, graceExpiration))
		return
	}

	if !cachedResult.graceValid() {
		// Not valid and not even within grace period: query, cache and return:
		return execOpAndCache()
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
	cachedResult.reloading = true // We'll be the one to do it
	cachedResult.reloadMu.Unlock()

	// reload in new goroutine.
	// Note: we're not using the return values, we're returning the cached (grace-valid) values.
	go execOpAndCache()

	return
}

// transformKey may arbitrarily transform long keys to short ones,
// saving time when storing them in the internal map.
//
// Saving space is not the only aspect though as shortening requires computation.
func transformKey(key string) string {
	// Hash key using SHA-1 if it's very long
	// to avoid storing long keys and also having to compare long keys in map lookups.
	if len(key) > 100 { // Arbitrary limit, a compromize between space-time (SHA-1 byte size is 20)
		checksum := sha1.Sum([]byte(key))
		key = string(checksum[:]) // A valid UTF-8 string is not required
	}

	return key
}

// opResult holds the result of an operation.
type opResult[T any] struct {
	expiresAt, graceExpiresAt time.Time

	result    T // If an op has multiple results, this should be a slice (e.g. []any)
	resultErr error

	reloadMu  sync.RWMutex
	reloading bool
}

// newOpResult creates a new OpResult.
func newOpResult[T any](result T, resultErr error, expiration, graceExpiration time.Duration) *opResult[T] {
	now := time.Now()
	return &opResult[T]{
		expiresAt:      now.Add(expiration),
		graceExpiresAt: now.Add(expiration + graceExpiration),
		result:         result,
		resultErr:      resultErr,
	}
}

// valid tells if the result is valid.
func (opr *opResult[T]) valid() bool {
	return opr != nil && time.Now().Before(opr.expiresAt)
}

// graceValid tells if the result is "grace-valid" (valid within the grace expiration beyond the normal expiration).
func (opr *opResult[T]) graceValid() bool {
	return opr != nil && time.Now().Before(opr.graceExpiresAt)
}
