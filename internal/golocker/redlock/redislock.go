package redlock

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/pyihe/gogame/internal/uuid"
	"github.com/pyihe/gogame/pkg"
)

const (
	// 式锁默认过期时间, 单位毫秒
	defaultLockExpire = 500

	lockCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
    return "OK"
else
    return redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
end`
	delCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return redis.call("GET", KEYS[1])
end`
)

type Executer interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	Ping(ctx context.Context) *redis.StatusCmd
}

func TryLock(executer Executer, key string, px int) (uint64, bool) {
	if px <= 0 {
		px = defaultLockExpire
	}

	lockId := uuid.Next()
	resp, err := executer.Eval(context.Background(), lockCommand, []string{key}, []string{strconv.FormatUint(lockId, 10), strconv.Itoa(px + defaultLockExpire)}).Result()
	if err == redis.Nil || resp == nil {
		return 0, false
	}
	if err != nil {
		return 0, false
	}
	reply := resp.(string)
	if reply == "OK" {
		return lockId, true
	}
	return 0, false
}

// Lock 阻塞式的获取锁
// key 用作分布式锁的键名称
// px锁的有效期，单位毫秒，超过此时间锁自动释放
// timeout 获取锁的超时时间，超过此时间如仍未获取到锁，则返回超时
func Lock(executer Executer, key string, px int, timeout time.Duration) (uint64, error) {
	if px <= 0 {
		px = defaultLockExpire
	}
	if timeout <= 0 {
		timeout = pkg.DefaultLockTimeout
	}

	var (
		lockId      = uuid.Next()
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	)

	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		default:
			resp, err := executer.Eval(context.Background(), lockCommand, []string{key}, []string{strconv.FormatUint(lockId, 10), strconv.Itoa(px + defaultLockExpire)}).Result()
			if err == redis.Nil || resp == nil {
				continue
			}
			if err != nil {
				return 0, err
			}
			reply := resp.(string)
			if reply == "OK" {
				return lockId, nil
			}
			time.Sleep(time.Microsecond)
		}
	}
}

// UnLock 解锁
func UnLock(executer Executer, key string, lockId uint64) error {
	for {
		resp, err := executer.Eval(context.Background(), delCommand, []string{key}, []string{strconv.FormatUint(lockId, 10)}).Result()
		// 解锁前未上锁
		if err == redis.Nil {
			panic("unlock of unlocked mutex")
		}
		if err != nil {
			return err
		}

		var reply int64
		switch resp.(type) {
		case int64:
			reply = resp.(int64)
		case string:
			reply, _ = strconv.ParseInt(resp.(string), 10, 64)
		}
		// 解锁成功
		if reply == 1 {
			return nil
		}
		// reply为其他值证明锁还未被其他持有者释放，继续等待
	}
}
