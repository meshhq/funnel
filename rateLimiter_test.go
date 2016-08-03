package gohttp

import (
	"flag"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/meshhq/meshCore/lib/gotils"
	"github.com/meshhq/meshCore/lib/meshRedis"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RateLimiterTest struct{}

var live = flag.Bool("redis", false, "Include redis tests")

var _ = Suite(&RateLimiterTest{})

func (r *RateLimiterTest) SetUpSuite(c *C) {
	err := meshRedis.SetupRedis()
	c.Assert(err, Equals, nil)
}

func (r *RateLimiterTest) TearDownSuite(c *C) {
	err := meshRedis.ClosePool()
	c.Assert(err, Equals, nil)
}

func (r *RateLimiterTest) SetUpTest(c *C) {
	// Nada for now
}

func (r *RateLimiterTest) TearDownTest(c *C) {
	// Nada for now
}

//---------
// Test A Successful Block
//---------

// TestSuccessfulRateLimiting tests a successful case where the rate limiter
// is provided a max window of 20Req/Sec. It is testing the approximate delay
// and completion of the entrance into all the dispatched routines
func (r *RateLimiterTest) TestSuccessfulRateLimiting(c *C) {

	// Max 10 Entires per second
	limiterInfo := &RateLimitInfo{
		Token:        "uniqueToken",
		MaxRequests:  20,
		TimeInterval: 1000,
	}

	pool := meshRedis.UnderlyingPool()
	rateLimiter := NewLimiter(limiterInfo, pool)

	// Tracking Begin Time
	beginTime := gotils.UnixInMilliseconds()

	// Sync the outcome
	var wg sync.WaitGroup

	var successCount uint64
	var totalCount uint64 = 41
	var i uint64
	for ; i < totalCount; i++ {
		// Increment the waitgroup by each event
		wg.Add(1)

		// Dispath all of these asynchronously
		go func() {
			defer wg.Done()
			// Attempt to enter the group
			err := rateLimiter.Enter()
			atomic.AddUint64(&successCount, 1)
			c.Assert(err, IsNil)
		}()

	}

	// Wait here until we're done
	wg.Wait()

	// Match the counts to make sure all completed
	c.Assert(successCount, Equals, totalCount)

	// Tracking End Time
	// This should be slightly over 2 seconds
	endTime := gotils.UnixInMilliseconds()

	totalTime := endTime - beginTime
	c.Assert(totalTime > 2000, Equals, true)
	c.Assert(totalTime < 3000, Equals, true)
}

// TestSuccessfulRateLimitingWithHigherNumOfOps tests a successful case as above
// but the number of concurrent operations is signifacantly more
func (r *RateLimiterTest) TestSuccessfulRateLimitingWithHigherNumOfOps(c *C) {

	// Max 10 Entires per second
	limiterInfo := &RateLimitInfo{
		Token:        "uniqueToken",
		MaxRequests:  500,
		TimeInterval: 1000,
	}

	rateLimiter := NewLimiter(limiterInfo, meshRedis.UnderlyingPool())

	// Tracking Begin Time
	beginTime := gotils.UnixInMilliseconds()

	// Sync the outcome
	var wg sync.WaitGroup

	var successCount uint64
	var totalCount uint64 = 1001
	var i uint64
	for ; i < totalCount; i++ {
		// Increment the waitgroup by each event
		wg.Add(1)

		// Dispath all of these asynchronously
		go func() {
			defer wg.Done()
			// Attempt to enter the group
			err := rateLimiter.Enter()
			atomic.AddUint64(&successCount, 1)
			c.Assert(err, IsNil)
		}()

	}

	// Wait here until we're done
	wg.Wait()

	// Match the counts to make sure all completed
	c.Assert(successCount, Equals, totalCount)

	// Tracking End Time
	// This should be slightly over 2 seconds
	endTime := gotils.UnixInMilliseconds()
	totalTime := endTime - beginTime
	c.Assert(totalTime > 2000, Equals, true)
	c.Assert(totalTime < 3000, Equals, true)
}

// TestRateLimitingDoesNotExceedRequestsInATimeInterval tests that all the requests haven't
// executed earlier than the allowed limit
func (r *RateLimiterTest) TestRateLimitingDoesNotExceedRequestsInATimeInterval(c *C) {

	// Max 10 Entires per second
	limiterInfo := &RateLimitInfo{
		Token:        "uniqueToken",
		MaxRequests:  10,
		TimeInterval: 1000,
	}

	rateLimiter := NewLimiter(limiterInfo, meshRedis.UnderlyingPool())

	// Sync the outcome
	var successCount uint64
	var totalCount uint64 = 20
	var i uint64
	for ; i < totalCount; i++ {
		// Dispath all of these asynchronously
		go func() {
			// Attempt to enter the group
			err := rateLimiter.Enter()
			atomic.AddUint64(&successCount, 1)
			c.Assert(err, IsNil)
		}()

	}

	time.Sleep(time.Duration(1) * time.Second)

	// Match the counts to make sure all completed
	c.Assert(successCount < totalCount, Equals, true)
}
