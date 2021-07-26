package util

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

// LockerProvider
var LockerProvider = wire.NewSet(NewLocker, wire.Bind(new(ILocker), new(*Locker)))

// ILocker
type ILocker interface {
	tryLock(ctx context.Context) (ok bool, err error)
}

// Locker
type Locker struct {
	resource string
	token    string
	client   redis.UniversalClient
	timeout  int
}

// NewLocker
func NewLocker(resource, token string,
	client redis.UniversalClient, timeout int) *Locker {
	return &Locker{
		resource: resource,
		token:    token,
		client:   client,
		timeout:  timeout,
	}
}

// TryLock
func TryLock(ctx context.Context, client redis.UniversalClient, resource string, token string, timeout int) (lock *Locker, ok bool, err error) {
	lock = &Locker{resource, token, client, timeout}
	ok, err = lock.tryLock(ctx)

	if !ok || err != nil {
		lock = nil
	}

	return
}

// tryLock
func (lock *Locker) tryLock(ctx context.Context) (ok bool, err error) {
	err = lock.client.Do(ctx, "SET", lock.key(), lock.token, "EX", int(lock.timeout), "NX").Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unlock
func (lock *Locker) Unlock(ctx context.Context) (err error) {
	err = lock.client.Do(ctx, "del", lock.key()).Err()
	return
}

// key
func (lock *Locker) key() string {
	return fmt.Sprintf("redislock:%s", lock.resource)
}

// AddTimeout
func (lock *Locker) AddTimeout(ctx context.Context, exTime int64) (ok bool, err error) {
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
