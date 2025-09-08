package observability

import (
	"fmt"

	"github.com/aws/aws-xray-sdk-go/xraylog"
	"github.com/spacelift-io/spcontext"
)

type xrayLogger struct {
	ctx   *spcontext.Context
	level xraylog.LogLevel
}

func NewXrayLogger(ctx *spcontext.Context, level xraylog.LogLevel) xraylog.Logger {
	return &xrayLogger{ctx: ctx, level: level}
}

func (logger *xrayLogger) Log(level xraylog.LogLevel, msg fmt.Stringer) {
	if level < logger.level {
		return
	}

	switch level {
	case xraylog.LogLevelDebug:
		logger.ctx.Debugf("%s", msg.String())
	case xraylog.LogLevelInfo:
		logger.ctx.Infof("%s", msg.String())
	case xraylog.LogLevelWarn:
		logger.ctx.Warnf("%s", msg.String())
	case xraylog.LogLevelError:
		logger.ctx.Errorf("%s", msg.String())
	default:
		logger.ctx.Errorf("%s", msg.String())
	}
}
