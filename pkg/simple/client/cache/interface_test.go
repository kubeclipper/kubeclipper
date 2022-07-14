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

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func CacheCommonTest(t *testing.T, kv Interface) {
	const (
		key   = "foo"
		value = "bar"
	)

	exist, err := kv.Exist(key)
	if assert.NoError(t, err) {
		assert.False(t, exist)
	}

	err = kv.Set(key, value, time.Second*5)
	assert.NoError(t, err)

	exist, err = kv.Exist(key)
	if assert.NoError(t, err) {
		assert.True(t, exist)
	}

	getValue, err := kv.Get(key)
	if assert.NoError(t, err) {
		assert.Equal(t, value, getValue)
	}

	err = kv.Expire(key, time.Second*10)
	assert.NoError(t, err)

	getValue, err = kv.Get(key)
	if assert.NoError(t, err) {
		assert.Equal(t, value, getValue)
	}

	err = kv.Remove(key)
	assert.NoError(t, err)

	_, err = kv.Get(key)
	assert.True(t, IsNotExists(err))
}
