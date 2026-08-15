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

package operationv2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const deterministicNameHashLength = 32

// TaskName is the idempotency key for one persisted execution attempt.
func TaskName(operationUID types.UID, retryGeneration int64, stepID string, nodeUID types.UID, attempt int32) string {
	return "task-" + stableHash(
		string(operationUID),
		fmt.Sprintf("%d", retryGeneration),
		stepID,
		string(nodeUID),
		fmt.Sprintf("%d", attempt),
	)
}

// LockName maps every target UID to exactly one etcd key.
func LockName(kind string, targetUID types.UID) string {
	prefix := strings.ToLower(kind)
	if prefix == "" {
		prefix = "target"
	}
	return prefix + "-" + stableHash(kind, string(targetUID))
}

func stableHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:deterministicNameHashLength]
}
