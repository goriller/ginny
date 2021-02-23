package utils

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Lock
type Lock struct {
	resource string
	token    string
	client   redis.UniversalClient
	timeout  int
}

// tryLock
func (lock *Lock) tryLock(ctx context.Context) (ok bool, err error) {
	err = lock.client.Do(ctx, "SET", lock.key(), lock.token, "EX", int(lock.timeout), "NX").Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unlock
func (lock *Lock) Unlock(ctx context.Context) (err error) {
	err = lock.client.Do(ctx, "del", lock.key()).Err()
	return
}

// key
func (lock *Lock) key() string {
	return fmt.Sprintf("redislock:%s", lock.resource)
}

// AddTimeout
func (lock *Lock) AddTimeout(ctx context.Context, exTime int64) (ok bool, err error) {
	ttlTime, err := lock.client.Do(ctx, "TTL", lock.key()).Int64()
	if err != nil {
		return false, err
	}
	if ttlTime > 0 {
		err := lock.client.Do(ctx, "SET", lock.key(), lock.token, "EX", int(ttlTime+exTime)).Err()
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// TryLock
func TryLock(ctx context.Context, client redis.UniversalClient, resource string, token string, timeout int) (lock *Lock, ok bool, err error) {
	lock = &Lock{resource, token, client, timeout}
	ok, err = lock.tryLock(ctx)

	if !ok || err != nil {
		lock = nil
	}

	return
}
