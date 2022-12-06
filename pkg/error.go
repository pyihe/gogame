package pkg

import "errors"

var (
	ErrMessageTooShort          = errors.New("message too short")
	ErrMessageTooLong           = errors.New("message too long")
	ErrMessageTooMany           = errors.New("too many messages")
	ErrCodecRequired            = errors.New("codec required")
	ErrPointerRequired          = errors.New("pointer required")
	ErrNotRegistered            = errors.New("not registered")
	ErrRepeatedRegister         = errors.New("already registered")
	ErrFunctionTypeNotSupported = errors.New("not supported function")
	ErrTooManyCalls             = errors.New("too many calls")
	ErrServerNotAttached        = errors.New("server not attached")
	ErrFullChannel              = errors.New("full channel")
	ErrServerClosed             = errors.New("server closed")
	ErrConnClosed               = errors.New("closed connection")
	ErrNilNewAgent              = errors.New("NewAgent required")
	ErrTimerClosed              = errors.New("timer closed")
	ErrInvalidCronExpr          = errors.New("invalid cron expr")
	ErrTaskCronClosed           = errors.New("task cron closed")
)
