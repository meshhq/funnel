package funnel

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/meshhq/funnel/Godeps/_workspace/src/github.com/hjr265/redsync.go/redsync"
	"github.com/meshhq/funnel/Godeps/_workspace/src/github.com/meshhq/meshLog"
	"github.com/meshhq/funnel/Godeps/_workspace/src/github.com/meshhq/meshRedis"
)

const (
	// defaultMaxRequest is the default max amount of requests allowed to take place within the default expiration
	defaultMaxRequests = 10

	// defaultExpiration is the default max time for the rate limit window
	defaultTimeInterval = 1000

	// defaultRetries is the amount of times to attempt to retry beginning a new window. We want this to be large incase a
	defaultRetries = 1000

	// defaultFactor is used to add randomness to the retry logic
	defaultFactor = 0.5
)

// RateLimitInfo is an inteface that provides the sufficient information to
// create a RateLimiter
type RateLimitInfo struct {
	// Token is the unique token that is used for tracking the limited request
	Token string

	// MaxRequests represents the maximum amount of requests that can occur over the given time period
	MaxRequests int

	// TimeInterval represents the time duration that the max requests can take place inside of
	TimeInterval int64
}

// RateLimiter controls the amount of concurrent requests from GoHttp. All time is in milliseconds
type RateLimiter struct {

	/**
	 * REDPOOL
	 */

	// Redpool is a a reference to a struct that vendors a redigo connection
	pool meshRedis.RedPool

	/**
	 * LOCK INFO
	 */

	// Token is the unique token that is used for tracking the
	// limited request
	token string

	/**
	 * REQUEST INFO
	 */

	// MaxRequests represents the maximum amount of requests that can
	// occur over the given time period
	maxRequestsForTimeInterval int

	// TimeInterval represents the time duration that the max requests
	// can take place inside of
	timeInterval int64

	/**
	 * RETRY / LOCK LOGIC
	 */

	// currentCount represents the amount of attempts to acquire a lock
	currentCount int16

	// redMutex is a ref a dist lock
	nodeLock *redsync.Mutex

	// retries represents the max amount of retires to begin
	// the window
	retries int

	// delay is the time to wait between retries to create or
	// enter a new window
	delay int64

	// factor is the factor used to change the retry attempt
	factor float64

	// mutex is the local used to avoid crowding redlock
	mutex *sync.Mutex
}

// NewLimiter is a factory method for creating a rate limiter
func NewLimiter(limitInfo *RateLimitInfo) (*RateLimiter, error) {
	pool := meshRedis.UnderlyingPool()
	if pool == nil {
		return nil, fmt.Errorf("Failed to acquire Redis pool. Check that meshRedis is connected.")
	}

	// Append additional string on tag
	limiterToken := limitInfo.Token + "_rateLimiterToken"
	limiter := &RateLimiter{
		token:                      limiterToken,
		timeInterval:               limitInfo.TimeInterval,
		maxRequestsForTimeInterval: limitInfo.MaxRequests,
		delay: limitInfo.TimeInterval / 4,
	}
	limiter.mutex = &sync.Mutex{}
	limiter.pool = pool
	return limiter, nil
}

// Enter attempts to enter the request into the current pool
func (r *RateLimiter) Enter() error {

	// Set expiration
	timeInterval := r.timeInterval
	if timeInterval == 0 {
		timeInterval = defaultTimeInterval
	}

	// Set retries
	retries := r.retries
	if retries == 0 {
		retries = defaultRetries
	}

	// Set delay
	delay := r.delay
	if delay == 0 {
		delay = delay / 10.0
	}

	// Set factor
	factor := r.factor
	if factor == 0 {
		factor = defaultFactor
	}

	// Local token ref
	token := r.rateLimiterToken()

	// Begin process of trying to enter into the
	// current window on this process. Lock across this
	// process to avoid rushing redis
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Lock this job across processes too, but only after a
	// sequential local lock
	redMutex := r.redMutexForTask(factor, delay)
	err := redMutex.Lock()
	if err != nil {
		meshLog.Fatalf("Error acquiring local redlock on ratelimiter with error: %+v", token)
		return err
	}
	defer redMutex.Unlock()

	// Create and close the redis connection we were handed
	redisSession := meshRedis.NewSessionWithExistingPool(r.pool)
	defer redisSession.CloseSession()

	// Enter a loop to begin the tries to enter the limiter group
	for i := 0; i < retries; i++ {
		// First try to resolve the list and get a count
		count, err := redisSession.GetListCount(token)
		if err != nil {
			meshLog.Fatal(err)
		}

		if err != nil || count >= r.maxRequestsForTimeInterval {
			// Sleep w/ a randomness factor
			sleepTime := (rand.Float64() * factor * float64(delay)) + float64(delay)
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		} else {
			// The key doesnt exists, or we're below our limit
			//
			// Check for the key existence
			exists, err := redisSession.KeyExists(token)
			if err != nil {
				meshLog.Fatal(err)
				continue
			}

			// If key doesn't exist, push it w/ expiration
			if !exists {
				// Multi cmd
				err = redisSession.AtomicPushOnListWithMsExpiration(token, token, timeInterval)
				if err != nil {
					meshLog.Fatalf("Block Creation Error In Rate Limiter: %+v", err)
					continue
				}
			} else {
				// RPush
				_, err = redisSession.RPushX(token, token)
				if err != nil {
					meshLog.Fatal(err)
					continue
				}
			}
			// Success! Let's return w/ no error
			return nil
		}
	}

	return errors.New("Unable to process request. Max attempts hit in the Rate Limiter")
}

/**
 * Tokens
 */

// redlockTokenForToken is the token used for redlock
func (r *RateLimiter) redlockToken() string {
	return r.token + "_redSync"
}

// redlockTokenForToken is the token used for redlock
func (r *RateLimiter) rateLimiterToken() string {
	return r.token + "_rateLimiterToken"
}

/**
 * Red Lock Mutex
 */

// redMutexForTask vendors a configured redlock w/ randomness builtin for the expiration
// of the lock
func (r *RateLimiter) redMutexForTask(factor float64, delay int64) *redsync.Mutex {
	// Grab the pool
	redisPool := r.pool
	nodes := []redsync.Pool{redisPool}

	// Generate the mutex w/ token
	redSyncToken := r.redlockToken()
	redMutex, err := redsync.NewMutexWithGenericPool(redSyncToken, nodes)
	if err != nil {
		meshLog.Fatalf("Error creating RedMutex in limiter: %+v", err)
		return nil
	}

	// Configure the mutex to have add sleep time randomness to its waiting. It was
	// found in testing that w/ out this, the system locks in step w/ itself when not using
	// a local pmutex. This is a danger for dist systems
	redMutex.Tries = 10000
	sleepTime := (rand.Float64() * factor * float64(delay)) + float64(delay)
	redMutex.Delay = time.Duration(sleepTime) * time.Millisecond
	redMutex.Expiry = 15 * time.Second
	return redMutex
}
