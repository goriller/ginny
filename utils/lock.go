package utils

import (
	"fmt"
	"log"

	"github.com/gomodule/redigo/redis"
)

// Lock
type Lock struct {
	resource string
	token    string
	conn     redis.Conn
	timeout  int
}

// tryLock
func (lock *Lock) tryLock() (ok bool, err error) {
	_, err = redis.String(lock.conn.Do("SET", lock.key(), lock.token, "EX", int(lock.timeout), "NX"))
	if err == redis.ErrNil {
		// The lock was not successful, it already exists.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unlock
func (lock *Lock) Unlock() (err error) {
	_, err = lock.conn.Do("del", lock.key())
	return
}

// key
func (lock *Lock) key() string {
	return fmt.Sprintf("redislock:%s", lock.resource)
}

// AddTimeout
func (lock *Lock) AddTimeout(exTime int64) (ok bool, err error) {
	ttlTime, err := redis.Int64(lock.conn.Do("TTL", lock.key()))
	if err != nil {
		log.Fatal("redisdb get failed:", err)
	}
	if ttlTime > 0 {
		_, err := redis.String(lock.conn.Do("SET", lock.key(), lock.token, "EX", int(ttlTime+exTime)))
		if err == redis.ErrNil {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// TryLock
func TryLock(conn redis.Conn, resource string, token string, timeout int) (lock *Lock, ok bool, err error) {
	lock = &Lock{resource, token, conn, timeout}
	ok, err = lock.tryLock()

	if !ok || err != nil {
		lock = nil
	}

	return
}
