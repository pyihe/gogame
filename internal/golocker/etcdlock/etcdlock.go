package etcdlock

import (
	"context"
	"time"

	"github.com/pyihe/gogame/pkg"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const (
	tolerance         = 1
	defaultLockExpire = 60
)

type Mutex struct {
	m *concurrency.Mutex
}

// NewMutex 新建一把锁
func NewMutex(client *clientv3.Client, key string, ttlSec int) (*Mutex, error) {
	if ttlSec <= 0 {
		ttlSec = defaultLockExpire
	}
	session, err := concurrency.NewSession(client, concurrency.WithTTL(ttlSec+tolerance))
	if err != nil {
		return nil, err
	}

	return &Mutex{concurrency.NewMutex(session, key)}, err
}

func (mutex *Mutex) Key() string {
	return mutex.m.Key()
}

func (mutex *Mutex) Lock(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = pkg.DefaultLockTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return mutex.m.Lock(ctx)
}

func (mutex *Mutex) TryLock(ctx context.Context) error {
	return mutex.m.TryLock(ctx)
}

func (mutex *Mutex) Unlock() error {
	return mutex.m.Unlock(context.Background())
}
