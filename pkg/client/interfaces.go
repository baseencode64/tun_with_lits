//go:generate mockgen -destination=mocks/client_mocks.go -source=interfaces.go -package=mocks -typed

package client

import (
	"context"
	"io"

	"github.com/goxray/core/network/route"
	xcommon "github.com/xtls/xray-core/common"
)

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)
// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

=======
// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

>>>>>>> 8374170 (feat: add automatic server selection from raw VLESS lists)
type pipe interface {
	Copy(ctx context.Context, pipe io.ReadWriteCloser, socks5 string) error
}

type ipTable interface {
	// Add adds route to ip table.
	Add(options route.Opts) error
	// Delete deletes route from ip table.
	Delete(options route.Opts) error
}

type runnable interface {
	xcommon.Runnable
}

//nolint:unused
type ioReadWriteCloser interface {
	io.ReadWriteCloser
}
