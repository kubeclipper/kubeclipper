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
	"context"
	"errors"
	"fmt"
	"time"

	redisV8 "github.com/go-redis/redis/v8"
)

func NewRedis(opt *RedisOptions) (Interface, error) {
	if len(opt.Addrs) == 0 {
		return nil, fmt.Errorf("redis addresses cannot be empty")
	}

	kv := redisKV{
		prefix: opt.Prefix,
	}

	switch opt.Schema {
	case Redis:
		kv.client = redisV8.NewClient(&redisV8.Options{
			Addr:     opt.Addrs[0],
			Username: opt.Username,
			Password: opt.Password,
			DB:       opt.DB,
		})
	case RedisSentinel:
		kv.client = redisV8.NewFailoverClient(&redisV8.FailoverOptions{
			SentinelAddrs: opt.Addrs,
			Username:      opt.Username,
			Password:      opt.Password,
			DB:            opt.DB,
		})
	case RedisCluster:
		kv.client = redisV8.NewClusterClient(&redisV8.ClusterOptions{
			Addrs:    opt.Addrs,
			Username: opt.Username,
			Password: opt.Password,
		})
	default:
		return nil, fmt.Errorf("not support redis schema:%s", opt.Schema)
	}

	return &kv, nil
}

var _ Interface = &redisKV{}

type redisKV struct {
	client redisV8.Cmdable
	prefix string
}

func (r *redisKV) withPrefix(key string) string {
	return r.prefix + key
}

func (r *redisKV) Set(key, value string, expire time.Duration) error {
	key = r.withPrefix(key)

	_, err := r.client.Set(context.TODO(), key, value, expire).Result()
	return err
}

func (r *redisKV) Update(key, value string) error {
	key = r.withPrefix(key)
	ctx := context.TODO()

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	return r.client.Set(context.TODO(), key, value, ttl).Err()
}

func (r *redisKV) Get(key string) (value string, err error) {
	ctx := context.TODO()
	key = r.withPrefix(key)

	value, err = r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redisV8.Nil) {
			return "", ErrNotExists
		}
		return "", err
	}
	return value, nil
}

func (r *redisKV) Exist(key string) (bool, error) {
	ctx := context.TODO()
	key = r.withPrefix(key)

	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (r *redisKV) Remove(key string) error {
	key = r.withPrefix(key)

	return r.client.Del(context.TODO(), key).Err()
}

func (r *redisKV) Expire(key string, expire time.Duration) error {
	key = r.withPrefix(key)

	return r.client.Expire(context.TODO(), key, expire).Err()
}
