/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type loggingT struct {
	l      *zap.Logger
	mu     sync.Mutex
	filter LogFilter
}

var _logging = defaultZapLogger()

func defaultZapLogger() *loggingT {
	opts := NewLogOptions()
	encode := convertZapLogEncode(opts.EncodeType)
	level := convertZapLogLevel(opts.Level)
	var multiWriteSyncer []zapcore.WriteSyncer
	if opts.ToStderr {
		multiWriteSyncer = append(multiWriteSyncer, os.Stderr)
	}
	core := zapcore.NewCore(newDefaultProductionLogEncoder(encode), zapcore.NewMultiWriteSyncer(multiWriteSyncer...), level)
	zl := zap.New(core)
	zl = zl.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))

	return &loggingT{
		l:      zl,
		mu:     sync.Mutex{},
		filter: nil,
	}
}

func ApplyZapLoggerWithOptions(opts *Options) {
	_logging.mu.Lock()
	defer _logging.mu.Unlock()
	var multiWriteSyncer []zapcore.WriteSyncer

	if !opts.ToStderr {
		lumberJackLogger := &lumberjack.Logger{
			Filename:   opts.LogFile,
			MaxSize:    opts.LogFileMaxSizeMB,
			MaxBackups: opts.MaxBackups,
			MaxAge:     opts.MaxAge,
			Compress:   opts.Compress,
			LocalTime:  opts.UseLocalTimeBack,
		}
		multiWriteSyncer = append(multiWriteSyncer, zapcore.Lock(zapcore.AddSync(lumberJackLogger)))
	} else {
		multiWriteSyncer = append(multiWriteSyncer, os.Stderr)
	}
	encode := convertZapLogEncode(opts.EncodeType)
	level := convertZapLogLevel(opts.Level)
	core := zapcore.NewCore(newDefaultProductionLogEncoder(encode),
		zapcore.NewMultiWriteSyncer(multiWriteSyncer...),
		level)
	zl := zap.New(core)
	if level == zapcore.DebugLevel {
		// caller skip set 1
		// 使得 DEBUG 模式下 caller 的值为调用当前 package 的代码路径
		zl = zl.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		zl = zl.WithOptions(zap.AddStacktrace(zapcore.FatalLevel))
	}
	_logging.l = zl
}

func convertZapLogLevel(level string) zapcore.Level {
	var l zapcore.Level
	switch level {
	case "debug":
		l = zapcore.DebugLevel
	case "warn":
		l = zapcore.WarnLevel
	case "error":
		l = zapcore.ErrorLevel
	case "info":
		fallthrough
	default:
		l = zapcore.InfoLevel
	}
	return l
}

func convertZapLogEncode(encode string) EncodeType {
	switch encode {
	case "json", "JSON":
		return JSONEncode
	case "console", "Console", "CONSOLE":
		fallthrough
	default:
		return ConsoleEncode
	}
}

// lockAndFlushAll is like flushAll but locks l.mu first.
func (l *loggingT) lockAndFlushAll() {
	l.mu.Lock()
	l.flushAll()
	l.mu.Unlock()
}

func (l *loggingT) flushAll() {
	_ = l.l.Sync()
}

// LogFilter is a collection of functions that can filter all logging calls,
// e.g. for sanitization of arguments and prevent accidental leaking of secrets.
type LogFilter interface {
	Filter(args []interface{}) []interface{}
	FilterF(format string, args []interface{}) (string, []interface{})
}

func newDefaultProductionLogEncoder(encodeType EncodeType) zapcore.Encoder {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.Format("2006-01-02T15:04:05Z07:00"))
	}
	switch encodeType {
	case ConsoleEncode:
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(encCfg)
	case JSONEncode:
		fallthrough
	default:
		return zapcore.NewJSONEncoder(encCfg)
	}
}

func Info(msg string, fields ...zap.Field) {
	_logging.l.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	_logging.l.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	_logging.l.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	_logging.l.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	_logging.l.Fatal(msg, fields...)
}

func Infof(format string, args ...interface{}) {
	if _logging.filter != nil {
		format, args = _logging.filter.FilterF(format, args)
	}
	_logging.l.Info(fmt.Sprintf(format, args...))
}

func Debugf(format string, args ...interface{}) {
	if _logging.filter != nil {
		format, args = _logging.filter.FilterF(format, args)
	}
	_logging.l.Debug(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...interface{}) {
	if _logging.filter != nil {
		format, args = _logging.filter.FilterF(format, args)
	}
	_logging.l.Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	if _logging.filter != nil {
		format, args = _logging.filter.FilterF(format, args)
	}
	_logging.l.Error(fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	if _logging.filter != nil {
		format, args = _logging.filter.FilterF(format, args)
	}
	_logging.l.Fatal(fmt.Sprintf(format, args...))
}

func FlushLogs() {
	_logging.lockAndFlushAll()
}

func SetFilter(filter LogFilter) {
	_logging.mu.Lock()
	defer _logging.mu.Unlock()
	_logging.filter = filter
}

func ZapLogger(name string) *zap.Logger {
	return _logging.l.Named(name)
}

func WithName(name string) Logging {
	return Log{l: _logging.l.Named(name)}
}

type loggingKey struct{}

func IntoContext(ctx context.Context, l Logging) context.Context {
	return context.WithValue(ctx, loggingKey{}, l)
}

func FromContext(ctx context.Context) Logging {
	if v := ctx.Value(loggingKey{}); v != nil {
		return v.(Logging)
	}
	return WithName("unknown")
}

type Logging interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	WithName(name string) Logging
	WithFields(fields ...zap.Field) Logging
}

type Log struct {
	l *zap.Logger
}

func (l Log) WithFields(fields ...zap.Field) Logging {
	return Log{l: l.l.With(fields...)}
}

func (l Log) WithName(name string) Logging {
	return Log{
		l: l.l.Named(name),
	}
}

func (l Log) Debug(msg string, fields ...zap.Field) {
	l.l.Debug(msg, fields...)
}

func (l Log) Info(msg string, fields ...zap.Field) {
	l.l.Info(msg, fields...)
}

func (l Log) Warn(msg string, fields ...zap.Field) {
	l.l.Warn(msg, fields...)
}

func (l Log) Error(msg string, fields ...zap.Field) {
	l.l.Error(msg, fields...)
}

func (l Log) Fatal(msg string, fields ...zap.Field) {
	l.l.Fatal(msg, fields...)
}

func (l Log) Debugf(format string, args ...interface{}) {
	l.l.Debug(fmt.Sprintf(format, args...))
}

func (l Log) Infof(format string, args ...interface{}) {
	l.l.Info(fmt.Sprintf(format, args...))
}

func (l Log) Warnf(format string, args ...interface{}) {
	l.l.Warn(fmt.Sprintf(format, args...))
}

func (l Log) Errorf(format string, args ...interface{}) {
	l.l.Error(fmt.Sprintf(format, args...))
}

func (l Log) Fatalf(format string, args ...interface{}) {
	l.l.Fatal(fmt.Sprintf(format, args...))
}
