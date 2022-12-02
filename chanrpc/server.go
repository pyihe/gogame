package chanrpc

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/pyihe/gogame/pkg"
)

// RPC调用信息
type CallInfo struct {
	f          interface{}
	args       []interface{}
	resultChan chan *Result
	cb         interface{}
}

// RPC调用结果
type Result struct {
	// nil
	// interface{}
	// []interface{}
	value interface{}
	err   error
	// callback:
	// func(err error)
	// func(ret interface{}, err error)
	// func(ret []interface{}, err error)
	cb interface{}
}

// one client per goroutine (goroutine not safe)
type Server struct {
	// id -> function
	//
	// function:
	// func(args ...interface{})
	// func(args ...interface{}) interface{}
	// func(args ...interface{}) []interface{}
	closed int32

	mu        sync.RWMutex
	functions map[interface{}]interface{}

	chanCall chan *CallInfo
}

func NewServer(maxCall int) *Server {
	s := new(Server)
	s.functions = make(map[interface{}]interface{})
	s.chanCall = make(chan *CallInfo, maxCall)
	s.closed = pkg.StatusRunning
	return s
}

func (s *Server) setFunc(id interface{}, fn interface{}) {
	s.mu.Lock()
	s.functions[id] = fn
	s.mu.Unlock()
}

func (s *Server) getFunc(id interface{}) (fn interface{}) {
	s.mu.RLock()
	fn = s.functions[id]
	s.mu.RUnlock()
	return
}

func (s *Server) ret(ci *CallInfo, ri *Result) {
	if ci.resultChan == nil {
		return
	}

	ri.cb = ci.cb
	ci.resultChan <- ri
}

func (s *Server) call(request *CallInfo, block bool) (err error) {
	if atomic.LoadInt32(&s.closed) == pkg.StatusClosed {
		return pkg.ErrServerClosed
	}

	switch block {
	case true:
		s.chanCall <- request
	default:
		select {
		case s.chanCall <- request:
		default:
			err = pkg.ErrFullChannel
		}
	}
	return
}

func (s *Server) Close() {
	if !atomic.CompareAndSwapInt32(&s.closed, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	close(s.chanCall)

	for ci := range s.chanCall {
		s.ret(ci, &Result{
			err: pkg.ErrServerClosed,
		})
	}
}

func (s *Server) Chan() <-chan *CallInfo {
	return s.chanCall
}

// you must call the function before calling Open and Go
func (s *Server) Register(id interface{}, f interface{}) error {
	if atomic.LoadInt32(&s.closed) == pkg.StatusClosed {
		return pkg.ErrServerClosed
	}
	// 只支持一下三种类型的Function
	switch f.(type) {
	case func(...interface{}):
	case func(...interface{}) interface{}:
	case func(...interface{}) []interface{}:
	default:
		return pkg.ErrFunctionTypeNotSupported
	}

	// 一个ID只能注册一次
	if registeredF := s.getFunc(id); registeredF != nil {
		return pkg.ErrRepeatedRegister
	}

	s.setFunc(id, f)
	return nil
}

func (s *Server) Exec(callInfo *CallInfo) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, pkg.StackSize)
			n := runtime.Stack(buf, false)
			retInfo := &Result{err: fmt.Errorf("RPC: %v exec failed: %s", callInfo.f, buf[:n])}
			s.ret(callInfo, retInfo)
		}
	}()

	result := &Result{
		cb: callInfo.cb, // 将请求中的回调赋值给返回结果
	}

	switch callInfo.f.(type) {
	case func(...interface{}): // 没有返回值的函数
		fn := callInfo.f.(func(...interface{}))
		fn(callInfo.args...)
		if callInfo.resultChan != nil {
			callInfo.resultChan <- result
		}

	case func(...interface{}) interface{}:
		fn := callInfo.f.(func(...interface{}) interface{})
		result.value = fn(callInfo.args...)
		if callInfo.resultChan != nil {
			callInfo.resultChan <- result
		}

	case func(...interface{}) []interface{}:
		fn := callInfo.f.(func(...interface{}) []interface{})
		result.value = fn(callInfo.args...)
		if callInfo.resultChan != nil {
			callInfo.resultChan <- result
		}

	default:
		panic(fmt.Sprintf("unsupported call function: %v", callInfo))
	}
}

// goroutine safe
func (s *Server) Go(id interface{}, args ...interface{}) {
	f := s.getFunc(id)
	if f != nil {
		s.chanCall <- &CallInfo{
			f:    f,
			args: args,
		}
	}
}

// goroutine safe
func (s *Server) Call0(id interface{}, args ...interface{}) (err error) {
	client := s.Open(0)
	err = client.Call0(id, args...)
	client.Close()
	return
}

// goroutine safe
func (s *Server) Call1(id interface{}, args ...interface{}) (result interface{}, err error) {
	client := s.Open(0)
	result, err = client.Call1(id, args...)
	client.Close()
	return
}

// goroutine safe
func (s *Server) CallN(id interface{}, args ...interface{}) (result []interface{}, err error) {
	client := s.Open(0)
	result, err = client.CallN(id, args...)
	client.Close()
	return
}

// goroutine safe
func (s *Server) Open(size int) *Client {
	c := NewClient(size)
	c.AttachSever(s)
	return c
}
