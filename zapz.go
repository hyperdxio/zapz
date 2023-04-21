package zapz

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/hyperdxio/hyperdx-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogzTimeEncoder format to time.RFC3339Nano
func LogzTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format(time.RFC3339Nano))
}

// DefaultConfig - Message needs to be the message key for hyperdx
var DefaultConfig = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "message",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     LogzTimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

const defaultType = "zap-logger"

// Zapz struct for logging
type Zapz struct {
	lz    *hyperdx.HyperdxSender
	level zapcore.Level
	enCfg zapcore.EncoderConfig
	typ   string
}

func New(token string, opts ...Option) (*zap.Logger, error) {
	logz, err := hyperdx.New(token)
	if err != nil {
		return nil, err
	}
	return NewLogz(logz, opts...)
}

func NewLogz(logz *hyperdx.HyperdxSender, opts ...Option) (*zap.Logger, error) {
	z := &Zapz{
		lz:    logz,
		level: zap.InfoLevel,
		enCfg: DefaultConfig,
		typ:   defaultType,
	}

	if len(opts) > 0 {
		for _, v := range opts {
			v.apply(z)
		}
	}

	en := zapcore.NewJSONEncoder(z.enCfg)
	hostname, _ := os.Hostname()
	return zap.New(zapcore.NewCore(en, z.lz, z.level)).With(
		zap.String("type", z.typ),
		zap.String("__hdx_sv", os.Getenv("OTEL_SERVICE_NAME")),
		zap.String("__hdx_h", hostname),
	), nil
}

// An Option configures a Logger.
type Option interface {
	apply(z *Zapz)
}

// Helper to hook with otel trace
func WithTraceMetadata(ctx context.Context, logger *zap.Logger) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		// ctx does not contain a valid span.
		// There is no trace metadata to add.
		return logger
	}
	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
		zap.String("trace_flags", spanContext.TraceFlags().String()),
	)
}

// SetLevel set the log level
func SetLevel(l zapcore.Level) Option {
	return optionFunc(func(z *Zapz) {
		z.level = l
	})
}

// SetEncodeConfig set the encoder
func SetEncodeConfig(c zapcore.EncoderConfig) Option {
	return optionFunc(func(z *Zapz) {
		z.enCfg = c
	})
}

// SetLogz use this logzsender
func SetLogz(c *hyperdx.HyperdxSender) Option {
	return optionFunc(func(z *Zapz) {
		z.lz = c
	})
}

func SetUrl(url string) Option {
	return optionFunc(func(z *Zapz) {
		hyperdx.SetUrl(url)(z.lz)
	})
}

// SetType setting log type zap.Field
func SetType(ty string) Option {
	return optionFunc(func(z *Zapz) {
		z.typ = ty
	})
}

// WithDebug enables debugging output for log
func WithDebug(w io.Writer) Option {
	return optionFunc(func(z *Zapz) {
		hyperdx.SetDebug(w)(z.lz)
	})
}

type optionFunc func(z *Zapz)

func (f optionFunc) apply(z *Zapz) {
	f(z)
}
