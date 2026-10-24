/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package redact

import (
	"regexp"
	"strings"
)

const logValuePattern = `("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^",[:space:]}]+)`

var (
	keyValueSecretPattern = regexp.MustCompile(
		`(?i)("?(?:password|passwd|token|secret)"?[[:space:]]*[=:][[:space:]]*)` + logValuePattern,
	)
	authorizationPattern = regexp.MustCompile(
		`(?i)(authorization"?[[:space:]]*[=:][[:space:]]*)(?:bearer[[:space:]]+)?` + logValuePattern,
	)
	bearerPattern           = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]+=*`)
	jwtPattern              = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	brokerCredentialPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^@\s]+@`)
	pemPrivateKeyPattern    = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
	// A bounded capture may retain only one PEM marker. Hide the fragment as well.
	pemOpenStartPattern = regexp.MustCompile(`(?s)-{5}BEGIN [A-Z ]*PRIVATE KEY-{5}.*$`)
	pemOpenEndPattern   = regexp.MustCompile(`(?s)^.*?-{5}END [A-Z ]*PRIVATE KEY-{5}`)
)

// Text removes terminal controls and redacts recognized credential patterns.
func Text(value string) string {
	value = stripTerminalControls(value)
	value = pemPrivateKeyPattern.ReplaceAllString(value, "[REDACTED-PRIVATE-KEY]")
	value = pemOpenStartPattern.ReplaceAllString(value, "[REDACTED-PRIVATE-KEY]")
	value = pemOpenEndPattern.ReplaceAllString(value, "[REDACTED-PRIVATE-KEY]")
	value = redactValues(authorizationPattern, value)
	value = redactValues(keyValueSecretPattern, value)
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = jwtPattern.ReplaceAllString(value, "[REDACTED-JWT]")
	return brokerCredentialPattern.ReplaceAllString(value, "$1[REDACTED]@")
}

func redactValues(pattern *regexp.Regexp, value string) string {
	return pattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		replacement := "[REDACTED]"
		if quote := parts[2][0]; quote == '"' || quote == '\'' {
			replacement = string(quote) + replacement + string(quote)
		}
		return parts[1] + replacement
	})
}

func stripTerminalControls(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for i := 0; i < len(value); i++ {
		character := value[i]
		if character == '\x1b' {
			i = skipEscapeSequence(value, i)
			continue
		}
		if character < 0x20 && character != '\n' && character != '\t' || character == 0x7f {
			continue
		}
		result.WriteByte(character)
	}
	return result.String()
}

func skipEscapeSequence(value string, start int) int {
	if start+1 >= len(value) {
		return start
	}
	switch value[start+1] {
	case '[':
		for i := start + 2; i < len(value); i++ {
			if value[i] >= 0x40 && value[i] <= 0x7e {
				return i
			}
		}
		return len(value) - 1
	case ']':
		for i := start + 2; i < len(value); i++ {
			if value[i] == '\a' {
				return i
			}
			if value[i] == '\x1b' && i+1 < len(value) && value[i+1] == '\\' {
				return i + 1
			}
		}
		return len(value) - 1
	default:
		return start + 1
	}
}
