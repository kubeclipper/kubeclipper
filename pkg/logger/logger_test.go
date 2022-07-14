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
	"testing"

	"go.uber.org/zap"
)

func BenchmarkInfoWithConsoleEncode(b *testing.B) {
	defer FlushLogs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for pb.Next() {
				Info("test",
					zap.Int64("int64-1", int64(1)),
					zap.Int64("int64-2", int64(2)),
					zap.Float64("float64", 1.0),
					zap.String("string1", "\n"),
					zap.String("string2", "💩"),
					zap.String("string3", "🤔"),
					zap.String("string4", "🙊"),
					zap.Bool("bool", true),
					zap.Any("request", struct {
						Method  string `json:"method"`
						Timeout int    `json:"timeout"`
						secret  string
					}{
						Method:  "GET",
						Timeout: 10,
						secret:  "pony",
					}))
			}
		}
	})
	b.StopTimer()
}

func BenchmarkInfoWithJsonEncode(b *testing.B) {
	//_logging.encodeType = JSONEncode
	//ApplyLogger()
	defer FlushLogs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for pb.Next() {
				Info("test",
					zap.Int64("int64-1", int64(1)),
					zap.Int64("int64-2", int64(2)),
					zap.Float64("float64", 1.0),
					zap.String("string1", "\n"),
					zap.String("string2", "💩"),
					zap.String("string3", "🤔"),
					zap.String("string4", "🙊"),
					zap.Bool("bool", true),
					zap.Any("request", struct {
						Method  string `json:"method"`
						Timeout int    `json:"timeout"`
						secret  string
					}{
						Method:  "GET",
						Timeout: 10,
						secret:  "pony",
					}))
			}
		}
	})
	b.StopTimer()
}
