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
	"fmt"

	"github.com/spf13/pflag"
)

type Options struct {
	LogFile          string `json:"logFile" yaml:"logFile"`
	LogFileMaxSizeMB int    `json:"logFileMaxSizeMB" yaml:"logFileMaxSizeMB"`
	ToStderr         bool   `json:"toStderr" yaml:"toStderr"`
	Level            string `json:"level" yaml:"level"`
	EncodeType       string `json:"encodeType" yaml:"encodeType"`
	MaxBackups       int    `json:"maxBackups" yaml:"maxBackups"`
	MaxAge           int    `json:"maxAge" yaml:"maxAge"`
	Compress         bool   `json:"compress" yaml:"compress"`
	UseLocalTimeBack bool   `json:"useLocalTime" yaml:"useLocalTime"`
}

func NewLogOptions() *Options {
	return &Options{
		ToStderr:         true,
		LogFile:          "",
		LogFileMaxSizeMB: 100,
		Level:            "info",
		EncodeType:       "console",
		MaxAge:           30,
		MaxBackups:       5,
		Compress:         false,
		UseLocalTimeBack: true,
	}
}

func (s *Options) Validate() []error {
	if s == nil {
		return nil
	}

	var allErrors []error

	switch s.Level {
	case "debug", "info", "warn", "error":
	default:
		allErrors = append(allErrors, fmt.Errorf("--log-level must be one of debug,info,warn,error"))
	}

	switch s.EncodeType {
	case "json", "console":
	default:
		allErrors = append(allErrors, fmt.Errorf("--log-encode-type must be one of json or console"))
	}

	return allErrors
}

func (s *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.LogFile, "log-file", s.LogFile, "If non-empty, use this log file")
	fs.IntVar(&s.LogFileMaxSizeMB, "log-file-max-size", s.LogFileMaxSizeMB,
		"Defines the maximum size a log file can grow to. Unit is megabytes. "+
			"If the value is 0, the maximum file size is unlimited.")
	fs.BoolVar(&s.ToStderr, "logtostderr", s.ToStderr, "log to standard error instead of files")
	fs.StringVar(&s.Level, "log-level", s.Level, "the number of the log level verbosity")
	fs.StringVar(&s.EncodeType, "log-encode-type", s.EncodeType, "the number of the log encode type, console or json")
	fs.IntVar(&s.MaxBackups, "log-max-backups", s.MaxBackups, ""+
		"MaxBackups is the maximum number of old log files to retain."+
		"The default is to retain all old log files (though MaxAge may still cause them to get deleted.)")
	fs.IntVar(&s.MaxAge, "log-max-age", s.MaxAge, ""+
		"MaxAge is the maximum number of days to retain old log files based on the timestamp encoded in their filename. "+
		"Note that a day is defined as 24 hours and may not exactly correspond to calendar days due to daylight savings, "+
		"leap seconds, etc. The default is not to remove old log files based on age.")
	fs.BoolVar(&s.Compress, "log-compress", s.Compress, ""+
		"Compress determines if the rotated log files should be compressed using gzip. The default is not to perform compression.")
	fs.BoolVar(&s.UseLocalTimeBack, "log-use-localtime", s.UseLocalTimeBack, ""+
		"LocalTime determines if the time used for formatting the timestamps in backup files is the computer's local time. "+
		"false mean to use UTC time.")
}
