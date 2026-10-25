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
	"strings"
	"testing"
)

func TestTextRedactsExistingPatternsAndControls(t *testing.T) {
	input := "\x1b[31m\x1b]52;c;clipboard\a" + `password=hunter2 {"token":"abc.def"} Authorization:Bearer abcd-secret Bearer another-secret amqp://admin:secret@example:5672 eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature` + "\n-----BEGIN PRIVATE KEY-----\nprivate-key-body\n-----END PRIVATE KEY-----"
	result := Text(input)
	for _, secret := range []string{"hunter2", "abc.def", "abcd-secret", "another-secret", "admin:secret", "eyJhbGciOiJIUzI1NiJ9", "private-key-body", "\x1b", "clipboard"} {
		if strings.Contains(result, secret) {
			t.Errorf("secret/control %q survived: %s", secret, result)
		}
	}
	if !strings.Contains(result, "REDACTED") {
		t.Fatal("missing redaction marker")
	}
}

func TestTextRedactsCompleteSecretValues(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
	}{
		{"quoted spaces", `{"password": "alpha beta", "detail": "startup failed"}`, `{"password": "[REDACTED]", "detail": "startup failed"}`},
		{"key whitespace", `{"password" : "alpha-beta"}`, `{"password" : "[REDACTED]"}`},
		{"escaped quotes", `{"password": "alpha\" beta\\ gamma"}`, `{"password": "[REDACTED]"}`},
		{"single quotes", `passwd = 'alpha beta' detail=failed`, `passwd = '[REDACTED]' detail=failed`},
		{"escaped single quote", `passwd='alpha\' beta'`, `passwd='[REDACTED]'`},
		{"quoted delimiters", `{"token":"alpha, beta]}"}`, `{"token":"[REDACTED]"}`},
		{"quoted authorization", `{"Authorization" : "Basic YWRtaW46cGFzcw=="}`, `{"Authorization" : "[REDACTED]"}`},
		{"bearer authorization", `Authorization : Bearer alpha-beta`, `Authorization : [REDACTED]`},
		{"unquoted value", `secret = alpha-beta detail=failed`, `secret = [REDACTED] detail=failed`},
		{"unquoted apostrophe", `secret=alpha'beta detail=failed`, `secret=[REDACTED] detail=failed`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Text(tt.input); got != tt.want {
				t.Errorf("Text() = %q, want %q", got, tt.want)
			}
		})
	}
}
