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
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

var _logging = defaultLogging()

func defaultLogging() loggingT {
	return loggingT{
		mu:        sync.Mutex{},
		verbosity: 0,
		Colorful:  true,
		Caller:    false,
	}
}

func AddFlags(flags *pflag.FlagSet) {
	flags.Var(&_logging.verbosity, "v", "number for the log level verbosity")
	flags.BoolVar(&_logging.Colorful, "colorized", _logging.Colorful, "print colorized log")
	flags.BoolVar(&_logging.Caller, "caller", _logging.Caller, "print log with caller")
}

type severity int32 // sync/atomic int32

const (
	infoLog severity = iota
	warningLog
	errorLog
	fatalLog
	//numSeverity = 4
)

var severityName = []string{
	infoLog:    "INFO",
	warningLog: "WARNING",
	errorLog:   "ERROR",
	fatalLog:   "FATAL",
}

var severityColorFunc = []func(format string, a ...interface{}) string{
	infoLog:    color.MagentaString,
	warningLog: color.YellowString,
	errorLog:   color.RedString,
	fatalLog:   color.HiRedString,
}

type loggingT struct {
	mu        sync.Mutex
	verbosity Level // V logging level, the value of the -v flag/
	Colorful  bool
	Caller    bool
}

func (l *loggingT) writeTS(buf *bytes.Buffer) {
	t := time.Now().Format("2006-01-02T15:04:05Z07:00")
	buf.WriteString("[")
	if l.Colorful {
		t = color.HiCyanString(t)
	}
	buf.WriteString(t)
	buf.WriteString("]")
}

func (l *loggingT) writeSeverity(buf *bytes.Buffer, s severity) {
	buf.WriteString("[")
	if l.Colorful {
		buf.WriteString(severityColorFunc[s](severityName[s]))
	} else {
		buf.WriteString(severityName[s])
	}
	buf.WriteString("]")
}

func (l *loggingT) writerCaller(buf *bytes.Buffer) {
	if l.Caller {
		_, file, lineno, ok := runtime.Caller(4)
		var strim = "/"
		if ok {
			codeArr := strings.Split(file, strim)
			code := codeArr[len(codeArr)-1]
			src := strings.Replace(
				fmt.Sprintf("%s:%d", code, lineno), "%2e", ".", -1)
			buf.WriteString("[")
			buf.WriteString(color.BlueString(src))
			buf.WriteString("]")
		}
	}
}

func (l *loggingT) addHeader(buf *bytes.Buffer, s severity) {
	l.writeTS(buf)
	l.writeSeverity(buf, s)
	l.writerCaller(buf)
	buf.WriteString(" ")
}

func (l *loggingT) output(buf *bytes.Buffer, s severity) {
	_, _ = color.Output.Write(buf.Bytes())
	if s == fatalLog {
		trace := stacks(false)
		_, _ = color.Error.Write(trace)
		os.Exit(255)
	}
}

func (l *loggingT) printf(s severity, format string, args ...interface{}) {
	buf := &bytes.Buffer{}
	l.addHeader(buf, s)
	_, _ = fmt.Fprintf(buf, format, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(buf, s)
}

func (l *loggingT) println(s severity, args ...interface{}) {
	buf := &bytes.Buffer{}
	l.addHeader(buf, s)
	_, _ = fmt.Fprintln(buf, args...)
	l.output(buf, s)
}

// stacks is a wrapper for runtime.Stack that attempts to recover the data for all goroutines.
func stacks(all bool) []byte {
	// We don't know how big the traces are, so grow a few times if they don't fit. Start large, though.
	n := 10000
	if all {
		n = 100000
	}
	var trace []byte
	for i := 0; i < 5; i++ {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

// Level specifies a level of verbosity for V logs. *Level implements
// flag.Value; the -v flag is of type Level and should be modified
// only through the flag.Value interface.
type Level int32

func (l *Level) Type() string {
	return "int32"
}

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Get is part of the flag.Getter interface.
func (l *Level) Get() interface{} {
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return err
	}
	_logging.mu.Lock()
	defer _logging.mu.Unlock()
	_logging.verbosity.set(Level(v))
	return nil
}

type Logger interface {
	Enabled() bool
	V(level Level) Logger
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

var _ Logger = verbose{}

type verbose struct {
	enabled bool
}

func newVerbose(b bool) verbose {
	return verbose{enabled: b}
}

func (v verbose) Enabled() bool {
	return v.enabled
}

func (v verbose) V(level Level) Logger {
	if _logging.verbosity.get() >= level {
		return newVerbose(true)
	}
	return newVerbose(false)
}

func (v verbose) Info(args ...interface{}) {
	if v.enabled {
		_logging.println(infoLog, args...)
	}
}

func (v verbose) Infof(format string, args ...interface{}) {
	if v.enabled {
		_logging.printf(infoLog, format, args...)
	}
}

func (v verbose) Warn(args ...interface{}) {
	if v.enabled {
		_logging.println(warningLog, args...)
	}
}

func (v verbose) Warnf(format string, args ...interface{}) {
	if v.enabled {
		_logging.printf(warningLog, format, args...)
	}
}

func (v verbose) Error(args ...interface{}) {
	if v.enabled {
		_logging.println(errorLog, args...)
	}
}

func (v verbose) Errorf(format string, args ...interface{}) {
	if v.enabled {
		_logging.printf(errorLog, format, args...)
	}
}

func (v verbose) Fatal(args ...interface{}) {
	if v.enabled {
		_logging.println(fatalLog, args...)
	}
}

func (v verbose) Fatalf(format string, args ...interface{}) {
	if v.enabled {
		_logging.printf(fatalLog, format, args...)
	}
}

func Info(args ...interface{}) {
	_logging.println(infoLog, args...)
}

func Infof(format string, args ...interface{}) {
	_logging.printf(infoLog, format, args...)
}

func Warn(args ...interface{}) {
	_logging.println(warningLog, args...)
}

func Warnf(format string, args ...interface{}) {
	_logging.printf(warningLog, format, args...)
}

func Error(args ...interface{}) {
	_logging.println(errorLog, args...)
}

func Errorf(format string, args ...interface{}) {
	_logging.printf(errorLog, format, args...)
}

func Fatal(args ...interface{}) {
	_logging.println(fatalLog, args...)
}

func Fatalf(format string, args ...interface{}) {
	_logging.printf(fatalLog, format, args...)
}

func V(level Level) Logger {
	if _logging.verbosity.get() >= level {
		return newVerbose(true)
	}
	return newVerbose(false)
}
