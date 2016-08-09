Funnel
======
[![Build Status](https://travis-ci.org/meshhq/funnel.svg?branch=master)](https://travis-ci.org/meshhq/funnel)

Intro
-----

Funnel is a distributed rate limter for golang. The project depends on redis and a number of redis related libs in order to accomplish distributed limited access to a resource.

### Why do I need a rate limiter?
A common use for a rate limiter could be consuming commercial APIs or limiting access to a shared resource that has a connection threshold. If you are consuming an API from a rate limited service, and you have reached the capacity, your requests will fail. It's much more convenient to use a rate limiter on the client than to try and implement retry logic that just hammers the endpoint until you have access.

### Why wouldn't I use the [Rate Limiter](https://github.com/golang/go/wiki/RateLimiting) from the GoWiki?
*Funnel* is a distributed solution for limiting control to a resource, while the one provided in the wiki only works if the access is limited to one process. Their solution doesn't hold up for scenarios when you have more than one apps (processes) trying to reach a resource.

### How do I use funnel?
#### ENVs
*`REDIS_URL` ENV needs to be set* 

Funnel is packaged with a thin redis client wrapper, [MeshRedis](https://github.com/meshhq/meshRedis). This dependency sets up a connection to redis, and uses redis to coordinate the distributed locking and list inclusions.

If `REDIS_URL` is not found, it will defer to the common localhost address:
`redis://127.0.0.1:6379"`

#### Setup
Funnel needs three pieces of infomation to help you limit access to a resource:
- Max requests for a given time interval
- The duration (time interval) for the limiter
- A token to uniquely identify the limted resource
```go
import "github.com/meshhq/funnel"

func main() {
    limiterInfo := &funnel.RateLimitInfo{
            Token:        "uniqueToken",
            MaxRequests:  20,
            TimeInterval: 1000,
        }
    rateLimiter := funnel.NewLimiter(limiterInfo)    
}
```

#### Use
After funnel is initialized, all you have to do is call `Enter()`. If there is room in the limiter, `Enter()` will let the execution continue. If the limtiter has reached it's max capacity for a given time interval, it will block until the limiter has expired. 
```go
import "github.com/meshhq/funnel"

func main() {
    // Limiter is set to a max of 20 Requests / Second
    limiterInfo := &funnel.RateLimitInfo{
            Token:        "uniqueToken",
            MaxRequests:  20,
            TimeInterval: 1000,
        }
    rateLimiter := funnel.NewLimiter(limiterInfo)    

    // Send 41 requests
    for i := 0; i < 41; i++ {
        // Enter limiter
        rateLimiter.Enter()
        // (Make Request)
    }
    // At least two seconds will have elapsed when the loop finishes
    // There were never more than 20 requests per second
}
```

### Contributing
PRs are welcome, but will be rejected unless test coverage is updated
- [Taylor Halliday](https://github.com/tayhalla)
- [Kevin Coleman](https://github.com/kcoleman731)