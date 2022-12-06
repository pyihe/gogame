package gotask

import (
	"fmt"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var tc *TaskCron

func init() {
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{"192.168.1.77:2379"}})
	if err != nil {
		fmt.Println("client", err)
	}

	tc = New(client, "taskcron")
}

func TestTaskCron_Add(t *testing.T) {
	err := tc.Remove("key1", "value1")
	fmt.Println(err)
	select {}
}
