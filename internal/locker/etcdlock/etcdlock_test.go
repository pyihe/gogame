package etcdlock

import (
	"fmt"
	"sync"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	client *clientv3.Client
)

func init() {
	var err error
	client, err = clientv3.New(clientv3.Config{Endpoints: []string{"192.168.1.77:2379"}})
	if err != nil {
		fmt.Println("new client: ", err)
	}
}

func TestNewMutex(t *testing.T) {
	var wg sync.WaitGroup
	var count int
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := NewMutex(client, "lock_key", 60)
			if err != nil {
				fmt.Printf("new mutex err: %v\n", err)
				return
			}
			if err = m.Lock(0); err != nil {
				fmt.Printf("lock err: %v\n", err)
				return
			}
			count += 1

			if err = m.Unlock(); err != nil {
				fmt.Printf("unlock err: %v\n", err)
				return
			}
		}()
	}
	wg.Wait()
	fmt.Println("count: ", count)
}
