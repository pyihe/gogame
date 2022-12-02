package chanrpc

import (
	"runtime"
	"sync"

	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

var clientPool sync.Pool

type Client struct {
	s               *Server      // RPC服务
	chanSyncRet     chan *Result // 同步调用返回的结果
	ChanAsynRet     chan *Result // 异步调用返回的结果
	pendingAsynCall int          // 尚未返回的异步RPC数量
}

func NewClient(asynSize int) (c *Client) {
	v := clientPool.Get()
	if v == nil {
		c = &Client{}
	} else {
		c = v.(*Client)
	}
	c.chanSyncRet = make(chan *Result, 1)
	c.ChanAsynRet = make(chan *Result, asynSize)
	return
}

func (c *Client) release() {
	clientPool.Put(c)
}

func assert(i interface{}) []interface{} {
	switch i.(type) {
	case []interface{}:
		return i.([]interface{})
	default:
		return nil
	}
}

func (c *Client) AttachSever(s *Server) {
	c.s = s
}

func (c *Client) Call0(id interface{}, args ...interface{}) (err error) {
	if c.s == nil {
		return pkg.ErrServerNotAttached
	}

	f := c.s.getFunc(id)
	if f == nil {
		return pkg.ErrNotRegistered
	}

	request := &CallInfo{
		f:          f,
		args:       args,
		resultChan: c.chanSyncRet,
	}

	err = c.s.call(request, true)
	if err == nil {
		result := <-c.chanSyncRet
		err = result.err
	}
	return
}

func (c *Client) Call1(id interface{}, args ...interface{}) (value interface{}, err error) {
	if c.s == nil {
		return nil, pkg.ErrServerNotAttached
	}
	f := c.s.getFunc(id)
	if f == nil {
		return nil, pkg.ErrNotRegistered
	}

	request := &CallInfo{
		f:          f,
		args:       args,
		resultChan: c.chanSyncRet,
	}

	err = c.s.call(request, true)
	if err == nil {
		result := <-c.chanSyncRet
		value = result.value
		err = result.err
	}
	return
}

func (c *Client) CallN(id interface{}, args ...interface{}) (value []interface{}, err error) {
	if c.s == nil {
		return nil, pkg.ErrServerNotAttached
	}
	f := c.s.getFunc(id)
	if f == nil {
		return nil, pkg.ErrNotRegistered
	}

	request := &CallInfo{
		f:          f,
		args:       args,
		resultChan: c.chanSyncRet,
	}
	err = c.s.call(request, true)
	if err == nil {
		result := <-c.chanSyncRet
		value = assert(result.value)
		err = result.err
	}
	return
}

func (c *Client) asynCall(id interface{}, args []interface{}, cb interface{}) {
	result := &Result{}

	if c.s == nil {
		result.err = pkg.ErrServerNotAttached
		c.ChanAsynRet <- result
		return
	}
	f := c.s.getFunc(id)
	if f == nil {
		result.err = pkg.ErrNotRegistered
		c.ChanAsynRet <- result
		return
	}

	request := &CallInfo{
		f:          f,
		args:       args,
		resultChan: c.ChanAsynRet,
		cb:         cb,
	}
	err := c.s.call(request, false)
	if err != nil {
		result.err = err
		c.ChanAsynRet <- result
	}
}

func (c *Client) AsynCall(id interface{}, args ...interface{}) {
	if len(args) < 1 {
		panic("callback function required")
	}

	_args := args[:len(args)-1]
	cb := args[len(args)-1]

	switch cb.(type) {
	case func(error):
	case func(interface{}, error):
	case func([]interface{}, error):
	default:
		panic("unsupported callback type")
	}

	// too many calls
	if c.pendingAsynCall >= cap(c.ChanAsynRet) {
		execCb(&Result{err: pkg.ErrTooManyCalls, cb: cb})
		return
	}

	c.asynCall(id, _args, cb)
	c.pendingAsynCall++
}

func execCb(ri *Result) {
	if ri == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, pkg.StackSize)
			n := runtime.Stack(buf, false)
			log.Printf("RPC Client: %v callback failed: %s", ri.cb, buf[:n])
		}
	}()

	// execute
	switch ri.cb.(type) {
	case func(error):
		cb := ri.cb.(func(err error))
		cb(ri.err)

	case func(interface{}, error):
		cb := ri.cb.(func(interface{}, error))
		cb(ri.value, ri.err)

	case func([]interface{}, error):
		cb := ri.cb.(func([]interface{}, error))
		cb(assert(ri.value), ri.err)
	}
	return
}

func (c *Client) Cb(ri *Result) {
	c.pendingAsynCall--
	execCb(ri)
}

func (c *Client) Close() {
	for c.pendingAsynCall > 0 {
		c.Cb(<-c.ChanAsynRet)
	}
	c.release()
}

func (c *Client) Idle() bool {
	return c.pendingAsynCall == 0
}
