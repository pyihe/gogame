package redlock

import (
	"fmt"
	"sync"
	"testing"

	"github.com/go-redis/redis/v9"
	"github.com/pyihe/gogame/pkg/log"
)

var (
	client *redis.Client
)

func init() {
	client = redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "192.168.1.77:6379",
	})

}

func TestLock(t *testing.T) {
	var count int
	var waiter sync.WaitGroup
	for i := 0; i < 2; i++ {
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			id, err := Lock(client, "lock_key", 500, 0)
			if err != nil {
				log.Printf("lock err: %v", err)
				return
			}
			count += 1
			err = UnLock(client, "lock_key", id)
			if err != nil {
				log.Printf("unlock err: %v", err)
				return
			}
		}()
	}
	waiter.Wait()
	fmt.Println(count)
}
