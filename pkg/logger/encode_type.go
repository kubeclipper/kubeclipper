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
	"errors"
	"fmt"
)

var errUnmarshalNilEncodeType = errors.New("can't unmarshal a nil *EncodeType")

// A EncodeType is a logging priority. Higher levels are more important.
type EncodeType int8

const (
	// ConsoleEncode logs are typically voluminous, and are usually disabled in
	// production.
	ConsoleEncode EncodeType = iota + 1
	// JSONEncode is the default logging priority.
	JSONEncode

	//_minLevel = ConsoleEncode
	//_maxLevel = JSONEncode
)

// Set sets the level for the flag.Value interface.
func (l *EncodeType) Set(s string) error {
	return l.UnmarshalText([]byte(s))
}

// Get gets the level for the flag.Getter interface.
func (l *EncodeType) Get() interface{} {
	return *l
}

// MarshalText marshals the Level to text. Note that the text representation
// drops the -Level suffix (see example).
func (l EncodeType) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// String returns a lower-case ASCII representation of the log level.
func (l EncodeType) String() string {
	switch l {
	case ConsoleEncode:
		return "console"
	case JSONEncode:
		return "json"
	default:
		return fmt.Sprintf("EncodeType(%d)", l)
	}
}

// UnmarshalText unmarshals text to a level. Like MarshalText, UnmarshalText
// expects the text representation of a Level to drop the -Level suffix (see
// example).
//
// In particular, this makes it easy to configure logging levels using YAML,
// TOML, or JSON files.
func (l *EncodeType) UnmarshalText(text []byte) error {
	if l == nil {
		return errUnmarshalNilEncodeType
	}
	if !l.unmarshalText(text) && !l.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unrecognized level: %q", text)
	}
	return nil
}

func (l *EncodeType) unmarshalText(text []byte) bool {
	switch string(text) {
	case "console", "CONSOLE":
		*l = ConsoleEncode
	case "json", "JSON", "": // make the zero value useful
		*l = JSONEncode
	default:
		return false
	}
	return true
}
