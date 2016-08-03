Funnel
======
[![Build Status](https://travis-ci.org/meshhq/funnel.svg?branch=master)](https://travis-ci.org/meshhq/funnel)

Intro
-----

Funnel is a distributed rate limter for golang. The project depends on redis and a number of redis related libs in order to accomplish distributed limited access to a resource.

### Why do I need a rate limiter?
The most common use for a rate limiter is probably commercial APIs. If you are consuming an API from a rate limited service, and you have reached the capacity, your requests will fail. It's much more convenient to use a rate limiter on the client than to try and implement retry logic that just hammers the endpoint until you have access.

### Why wouldn't I use the [Rate Limiter](https://github.com/golang/go/wiki/RateLimiting) from the GoWiki?
*Funnel* is a distributed solution for limiting control to a resource, while the one provided in the wiki only works if the access is limited to one process. Their solution doesn't hold up for scenarios when you have more than one apps (processes) trying to reach a resource.

### Usage

#### Setup
Funnel needs three pieces of infomation to help you limit access to a resource:
- Max requests for a given time interval
- The duration (time interval) for the limiter
- A token to uniquely identify the limted resource
```go
import "github.com/garyburd/redigo/redis"

func main() {
    pool := &redis.Pool{}
    limiterInfo := &RateLimitInfo{
            Token:        "uniqueToken",
            MaxRequests:  20,
            TimeInterval: 1000,
        }
    rateLimiter := NewLimiter(limiterInfo, pool)    
}
```

#### Use
After funnel is initialized, all you have to do is call `Enter()`. If there is room in the limiter, `Enter()` will let the execution continue. If the limtiter has reached it's max capacity for a given time interval, it will block until the limiter has expired. 
```go
import "github.com/garyburd/redigo/redis"

func main() {
    pool := &redis.Pool{}
    // Limiter is set to a max of 20 Requests / Second
    limiterInfo := &RateLimitInfo{
            Token:        "uniqueToken",
            MaxRequests:  20,
            TimeInterval: 1000,
        }
    rateLimiter := NewLimiter(limiterInfo, pool)    

    // Send 41 requests
    for i := 0; i < 41; i++ {
        // Enter limiter
        rateLimiter.Enter()
        // (Make Request)
    }
    // Two seconds will have elapsed when the loop finishes
}
```

### Contributing
PRs are welcome, but will be rejected unless test coverage is updated
- [Taylor Halliday](https://github.com/tayhalla)
- [Kevin Coleman](https://github.com/kcoleman731)