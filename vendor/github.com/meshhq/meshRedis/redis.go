package meshRedis

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisSession models a connection to an underlying NoSQL persistence store.
type RedisSession struct {
	connection redis.Conn
}

// pool is the connection pool to the Redis instance
var pool *redis.Pool

// connection is the main connection to redis
var connection *redis.Conn

// RedPool is a interface for struct that can vend a connection to
// a redigo connection
type RedPool interface {
	Get() redis.Conn
}

//---------
// Redis Connection
//---------

// SetupRedis establishes a connection to the Redis instance at the provided
// url. If the connection attempt is unsuccessful, an error object
// will be returned describing the failure.
//
// @param url: The URL address to which the connection will be established.
// EX: redis://127.0.0.1:6379/200
//
// NOTE: the path '200' specifies the DB ID number. Use this to create seperate instances
func SetupRedis() error {
	redisURL := os.Getenv("REDIS_URL")

	// Try the local Redis URL if no URL is set by ENV
	if len(redisURL) == 0 {
		redisURL = "redis://127.0.0.1:6379"
	}

	pool = createNewConnectionPool(redisURL)
	conn := pool.Get()
	defer conn.Close()

	return pingRedis(conn, time.Now())
}

// ClosePool kills the entire connection pool to redis
func ClosePool() error {
	return pool.Close()
}

// UnderlyingPool exposes a reference to the underlying pool
func UnderlyingPool() *redis.Pool {
	return pool
}

// createNewConnectionPool is the internal setup for a redis connection pool
func createNewConnectionPool(redisURL string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     60,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			connection, err := redis.DialURL(redisURL)
			if err != nil {
				return nil, err
			}

			// Later, we want to secure redis, not now
			// 'Do' will call the auth
			// if _, err := c.Do("AUTH", password); err != nil {
			// 	c.Close()
			// 	return nil, err
			// }
			return connection, err
		},
		TestOnBorrow: pingRedis,
		Wait:         true}
}

// pingRedis is used internally to ping the connection
func pingRedis(connection redis.Conn, _ time.Time) error {
	_, err := connection.Do("PING")
	return err
}

// Ping is used internally to ping the connection
func (s *RedisSession) Ping() error {
	return pingRedis(s.connection, time.Time{})
}

// NewSession issues a new meshRedis ResisSession. This will be the
// interface that will be used to perform actions on redis
func NewSession() *RedisSession {
	connection := pool.Get()
	return &RedisSession{connection}
}

// CloseSession kills the RedisSession instance
func (s *RedisSession) CloseSession() error {
	return s.connection.Close()
}

//---------
// Session With Connections From A Different Pool
//---------

// NewSessionWithExistingPool is a convenience for using MeshRedis
// with a pool managed by another source
func NewSessionWithExistingPool(poolVendor RedPool) *RedisSession {
	connection := poolVendor.Get()
	return &RedisSession{connection}
}

//---------
// Redis Commands
//---------

// UpdateExpirationOfKey updates the expiration value of a key in redis
// If no key is found, the `error` return value will be non-nil
func (s *RedisSession) UpdateExpirationOfKey(key string, seconds int) error {
	val, err := s.connection.Do("EXPIRE", key, seconds)
	if err != nil {
		return err
	}

	if updateTime, ok := val.(int64); ok {
		time := int(updateTime)
		if time == 0 {
			errorMsg := fmt.Sprintf("That key does not exist")
			return errors.New(errorMsg)
		}
		return nil
	}

	errorMsg := fmt.Sprintf("Error updating the expiration")
	return errors.New(errorMsg)
}

// PTTLForKey returns the lifetime of the value associated with the key in ms
// If no key is found, the `error` return value will be non-nil
func (s *RedisSession) PTTLForKey(key string) (int, error) {
	ttl, err := s.connection.Do("PTTL", key)
	if err != nil || ttl == nil {
		return 0, err
	}

	if updateTime, ok := ttl.(int64); ok {
		return int(updateTime), err
	}

	errorMsg := fmt.Sprintf("Error processing the Key")
	return 0, errors.New(errorMsg)
}

// KeyExists checks if a given key exits in redis
func (s *RedisSession) KeyExists(key string) (bool, error) {
	val, rErr := s.connection.Do("EXISTS", key)
	found, err := redis.Bool(val, rErr)
	return found, err
}

// FlushAllKeys wipes out all keys in the current redis DB
func (s *RedisSession) FlushAllKeys() error {
	val, err := s.connection.Do("FLUSHDB")

	if err != nil {
		return err
	}

	errorMsg := "There was an error flushing the DB\n"
	if byteVal, ok := val.(string); ok {
		if string(byteVal) == "OK" {
			return nil
		}
		return errors.New(errorMsg)
	}

	return errors.New(errorMsg)
}

// Delete removes a key from redis
func (s *RedisSession) Delete(key string) error {
	_, err := s.connection.Do("DEL", key)
	return err
}

//---------
// String Commands
//---------

// SetString assigns the string to the supplied key in redis
func (s *RedisSession) SetString(key string, value string) error {
	_, err := s.connection.Do("SET", key, value)
	return err
}

// SetStringWithExpiration assigns the int to the supplied key in redis
func (s *RedisSession) SetStringWithExpiration(key string, seconds int, value string) error {
	_, err := s.connection.Do("SETEX", key, seconds, value)
	return err
}

// GetString retreives the value from the store.
func (s *RedisSession) GetString(key string) (value string, err error) {
	val, err := s.connection.Do("GET", key)

	if err != nil || val == nil {
		return "", err
	}

	if byteVal, ok := val.([]byte); ok {
		return string(byteVal), err
	}

	errorMsg := fmt.Sprintf("The value for the key %s is not a string", key)
	return "", errors.New(errorMsg)
}

//---------
// Int Commands
//---------

// SetInt assigns the int to the supplied key in redis
func (s *RedisSession) SetInt(key string, value int) error {
	_, err := s.connection.Do("SET", key, value)
	return err
}

// SetIntWithExpiration assigns the int to the supplied key in redis
func (s *RedisSession) SetIntWithExpiration(key string, seconds int, value int) error {
	_, err := s.connection.Do("SETEX", key, seconds, value)
	return err
}

// GetInt retreives the value from the store.
func (s *RedisSession) GetInt(key string) (value int, err error) {
	val, err := s.connection.Do("GET", key)
	if err != nil || val == nil {
		return 0, err
	}

	if byteVal, ok := val.([]byte); ok {
		strVal := string(byteVal)
		intVal, err := strconv.Atoi(strVal)
		return intVal, err
	}

	errorMsg := fmt.Sprintf("The value for the key %s is not a integer", key)
	return 0, errors.New(errorMsg)
}

//---------
// List Commands
//---------

// GetListCount returns the count of the list for the given key. If none
// exists, it returns 0
func (s *RedisSession) GetListCount(key string) (count int, err error) {
	val, rErr := s.connection.Do("LLEN", key)
	count, err = redis.Int(val, rErr)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// RPush appends a list in redis with the key
func (s *RedisSession) RPush(key string, value string) (count int, err error) {
	val, rErr := s.connection.Do("RPUSH", key, value)
	count, err = redis.Int(val, rErr)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// RPushX appends a list in redis with the key if it exists
func (s *RedisSession) RPushX(key string, value string) (count int, err error) {
	val, rErr := s.connection.Do("RPUSHX", key, value)
	count, err = redis.Int(val, rErr)
	if err != nil {
		return 0, err
	}
	return count, nil
}

//---------
// Multi / Pipelined Reqs
//---------

// AtomicPushOnListWithMsExpiration pushes a token, and key to a list with
// expiration of the list in terms of Milliseconds
func (s *RedisSession) AtomicPushOnListWithMsExpiration(key string, value string, msToExpire int64) error {
	err := s.connection.Send("MULTI")
	if err != nil {
		return err
	}
	defer s.connection.Do("EXEC")

	err = s.connection.Send("RPUSH", key, value)
	if err != nil {
		return err
	}

	err = s.connection.Send("PEXPIRE", key, msToExpire)
	if err != nil {
		return err
	}

	return err
}
