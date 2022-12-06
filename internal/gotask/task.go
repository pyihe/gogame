package gotask

import (
	"context"
	"encoding/json"
	"math"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
	"github.com/robfig/cron/v3"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	statusNormal   = 1 // 任务状态：正常
	statusDel      = 2 // 任务状态：删除
	execLockPrefix = "exec"
)

type (
	Handler func(key string, value interface{})

	task struct {
		Key     string      `json:"key,omitempty"`      // 任务key
		Value   interface{} `json:"value,omitempty"`    // 任务value
		EtcdKey string      `json:"etcd_key,omitempty"` // 任务etcd key
		Spec    string      `json:"spec,omitempty"`     // spec
		Status  uint8       `json:"status,omitempty"`   // 任务状态
		Timeout int         `json:"timeout,omitempty"`  // 任务执行超时时间
		Handler Handler     `json:"-"`                  // 任务handler
	}

	TaskCron struct {
		prefix     string
		client     *clientv3.Client
		parser     cron.Parser
		taskPool   sync.Pool
		mu         sync.RWMutex
		tasks      map[string]*task
		status     int32
		cancelFunc context.CancelFunc
	}
)

func New(client *clientv3.Client, prefix string) *TaskCron {
	ctx, cancel := context.WithCancel(context.Background())

	tc := &TaskCron{
		client: client,
		prefix: prefix,
		parser: cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		tasks:  map[string]*task{},
		taskPool: sync.Pool{New: func() any {
			return &task{}
		}},
		status:     pkg.StatusInitial,
		cancelFunc: cancel,
	}

	gopool.AddTask(func() {
		tc.watch(ctx)
	})
	return tc
}

func (tc *TaskCron) Close() {
	if !atomic.CompareAndSwapInt32(&tc.status, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	tc.cancelFunc()
}

func (tc *TaskCron) Add(spec, key, value string, taskTimeout int, handler Handler) (err error) {
	if tc.isClosed() {
		err = pkg.ErrTaskCronClosed
		return
	}

	etcdKey := tc.buildEtcdKey(key, value)
	oldTask, err := tc.getTask(etcdKey)
	if err != nil {
		return err
	}
	if oldTask != nil {
		return pkg.ErrRepeatedRegister
	}
	if taskTimeout <= 0 {
		taskTimeout = -1
	}

	schedule, err := tc.parser.Parse(spec)
	if err != nil {
		return err
	}

	now := time.Now()
	nextTime := schedule.Next(now)
	if nextTime.IsZero() {
		err = pkg.ErrInvalidCronExpr
		return
	}

	t := tc.newTask()
	t.Key = key
	t.Value = value
	t.EtcdKey = etcdKey
	t.Status = statusNormal
	t.Timeout = taskTimeout
	t.Handler = handler
	t.Spec = spec

	vBytes, err := json.Marshal(t)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rsp, err := tc.client.Grant(ctx, nextTime.Unix()-now.Unix())
	if err != nil {
		return
	}

	_, err = tc.client.Put(ctx, t.EtcdKey, string(vBytes), clientv3.WithLease(rsp.ID))
	if err == nil {
		tc.mu.Lock()
		tc.tasks[etcdKey] = t
		tc.mu.Unlock()
	}
	return
}

func (tc *TaskCron) Remove(key, value string) error {
	if tc.isClosed() {
		return pkg.ErrTaskCronClosed
	}
	etcdKey := tc.buildEtcdKey(key, value)
	t, err := tc.getTask(etcdKey)
	if err != nil {
		return err
	}
	if t == nil {
		return nil
	}

	t.Status = statusDel

	vBytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err = tc.client.Put(ctx, t.EtcdKey, string(vBytes)); err != nil {
		return err
	}

	_, err = tc.client.Delete(ctx, t.EtcdKey)
	if err == nil {
		tc.mu.Lock()
		delete(tc.tasks, etcdKey)
		tc.mu.Unlock()
	}
	return err
}

func (tc *TaskCron) UpdateTime(key, value, spec string) error {
	if tc.isClosed() {
		return pkg.ErrTaskCronClosed
	}
	schedule, err := tc.parser.Parse(spec)
	if err != nil {
		return err
	}

	now := time.Now()
	nextTime := schedule.Next(now)
	if nextTime.IsZero() {
		return pkg.ErrInvalidCronExpr
	}

	etcdKey := tc.buildEtcdKey(key, value)
	t, err := tc.getTask(etcdKey)
	if err != nil {
		return err
	}

	if t == nil {
		return nil
	}

	t.Spec = spec
	vBytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rsp, err := tc.client.Grant(ctx, nextTime.Unix()-now.Unix())
	if err != nil {
		return err
	}

	_, err = tc.client.Put(ctx, etcdKey, string(vBytes), clientv3.WithLease(rsp.ID))
	if err == nil {
		tc.mu.Lock()
		tc.tasks[etcdKey] = t
		tc.mu.Unlock()
	}
	return err
}

func (tc *TaskCron) isClosed() bool {
	return atomic.LoadInt32(&tc.status) == pkg.StatusClosed
}

func (tc *TaskCron) watch(ctx context.Context) {
	tc.status = pkg.StatusRunning
	watchChan := tc.client.Watch(ctx, tc.prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())

	for events := range watchChan {
		if err := events.Err(); err != nil {
			continue
		}

		for i := range events.Events {

			evt := events.Events[i]
			if evt.Type != mvccpb.DELETE {
				continue
			}

			var (
				t     *task
				value = evt.PrevKv.Value
			)

			if len(value) == 0 {
				continue
			}
			if err := json.Unmarshal(value, &t); err != nil || t == nil {
				continue
			}

			if t.Status == statusDel {
				continue
			}

			now := time.Now()
			schedule, _ := tc.parser.Parse(t.Spec)
			nextTime := schedule.Next(now)
			if !nextTime.IsZero() { // 需要再次执行
				rsp, err := tc.client.Grant(context.Background(), nextTime.Unix()-now.Unix())
				if err != nil {
					continue
				}
				_, err = tc.client.Put(context.Background(), t.EtcdKey, string(value), clientv3.WithLease(rsp.ID))
				if err != nil {
					continue
				}
			}
			gopool.AddTask(func() {
				tc.runTask(t)
			})
		}
	}
}

func (tc *TaskCron) runTask(t *task) {
	var (
		ctx = context.Background()
		ttl = int64(t.Timeout)
	)
	if ttl <= 0 {
		ttl = math.MaxInt32
	}

	rsp, err := tc.client.Grant(ctx, int64(t.Timeout))
	if err != nil {
		return
	}

	defer tc.client.Revoke(ctx, rsp.ID)

	execKey := path.Join(execLockPrefix, t.EtcdKey)
	txnRsp, err := tc.client.Txn(ctx).If(clientv3.Compare(clientv3.CreateRevision(execKey), "=", 0)).
		Then(clientv3.OpPut(execKey, "", clientv3.WithLease(rsp.ID))).Commit()
	if err != nil {
		return
	}

	// 没有抢到
	if !txnRsp.Succeeded {
		return
	}

	tc.mu.Lock()
	mTask := tc.tasks[t.EtcdKey]
	tc.mu.Unlock()

	if mTask == nil || mTask.Handler == nil {
		return
	}
	timer := time.NewTimer(time.Duration(ttl) * time.Second)
	finish := make(chan struct{})
	gopool.AddTask(func() {
		mTask.Handler(mTask.Key, mTask.Value)
		finish <- struct{}{}
	})
	select {
	case <-timer.C:
	case <-finish:
	}
	timer.Stop()
}

func (tc *TaskCron) buildEtcdKey(key, value string) string {
	return path.Join(tc.prefix, key, value)
}

func (tc *TaskCron) getTask(etcdKey string) (*task, error) {
	// 内存中是否存在
	tc.mu.RLock()
	t, ok := tc.tasks[etcdKey]
	tc.mu.RUnlock()
	if ok && t != nil {
		return t, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rsp, err := tc.client.Get(ctx, etcdKey)
	if err != nil {
		return nil, err
	}

	t = nil
	if len(rsp.Kvs) > 0 {
		err = json.Unmarshal(rsp.Kvs[0].Value, &t)
	}
	return t, err
}

func (tc *TaskCron) newTask() (t *task) {
	var ok bool
	t, ok = tc.taskPool.Get().(*task)
	if !ok {
		t = &task{}
	}
	return
}

func (tc *TaskCron) releaseTask(t *task) {
	if t == nil {
		return
	}
	*t = task{}
	tc.taskPool.Put(t)
}
