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
	"sync"
	"time"
)

type entry struct {
	expireAt time.Time
	value    string
}

type memoryKV struct {
	storage *sync.Map
}

func NewMemory() (Interface, error) {
	return &memoryKV{
		storage: &sync.Map{},
	}, nil
}

func (m *memoryKV) Set(key, value string, expire time.Duration) error {
	var expireAt time.Time
	if expire > NoExpiration {
		expireAt = time.Now().Add(expire)
	}
	m.storage.Store(key, entry{
		expireAt: expireAt,
		value:    value,
	})
	return nil
}

func (m *memoryKV) Update(key, value string) error {
	e, err := m.get(key)
	if err != nil {
		return err
	}
	e.value = value
	m.storage.Store(key, e)
	return nil
}

func (m *memoryKV) get(key string) (*entry, error) {
	v, ok := m.storage.Load(key)
	if !ok {
		return nil, ErrNotExists
	}
	e := v.(entry)

	// check expireAt
	if !e.expireAt.IsZero() {
		now := time.Now()
		if now.After(e.expireAt) {
			m.storage.Delete(key)
			return nil, ErrNotExists
		}
	}

	return &e, nil
}

func (m *memoryKV) Get(key string) (value string, err error) {
	e, err := m.get(key)
	if err != nil {
		return "", err
	}
	return e.value, nil
}

func (m *memoryKV) Exist(key string) (bool, error) {
	_, err := m.get(key)
	if err != nil {
		if IsNotExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *memoryKV) Remove(key string) error {
	m.storage.Delete(key)
	return nil
}

func (m *memoryKV) Expire(key string, expire time.Duration) error {
	e, err := m.get(key)
	if err != nil {
		return err
	}
	e.expireAt = time.Now().Add(expire)

	m.storage.Store(key, *e)
	return nil
}
