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
	"errors"
	"fmt"
	"time"
)

const NoExpiration time.Duration = 0

var ErrNotExists = fmt.Errorf("key not exists")

type Interface interface {
	Set(key, value string, expire time.Duration) error
	Update(key, newValue string) error
	Get(key string) (value string, err error)
	Exist(key string) (bool, error)
	Remove(key string) error
	Expire(key string, expire time.Duration) error
}

func IsNotExists(e error) bool {
	return errors.Is(e, ErrNotExists)
}
