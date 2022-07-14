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
	"time"

	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

func NewEtcd(reader iam.TokenReader, writer iam.TokenWriter) (Interface, error) {
	return &etcdKV{
		reader: reader,
		writer: writer,
	}, nil
}

var _ Interface = &etcdKV{}

type etcdKV struct {
	reader iam.TokenReader
	writer iam.TokenWriter
}

func (e *etcdKV) Set(key, value string, expire time.Duration) error {
	ctx := context.TODO()
	_, err := e.writer.CreateToken(ctx, &iamv1.Token{
		TypeMeta: metav1.TypeMeta{
			Kind:       iamv1.KindToken,
			APIVersion: iamv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key,
		},
		Spec: iamv1.TokenSpec{
			TokenType: iamv1.TemporaryToken,
			Token:     value,
			TTL:       newTTL(expire),
		},
	})
	return err
}

func (e *etcdKV) Update(key, value string) error {
	ctx := context.TODO()
	token, err := e.reader.GetTokenEx(ctx, key, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return ErrNotExists
		}
		return err
	}
	token.Spec.Token = value
	_, err = e.writer.UpdateToken(ctx, token)
	return err
}

func (e *etcdKV) Get(key string) (value string, err error) {
	token, err := e.reader.GetTokenEx(context.TODO(), key, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return "", ErrNotExists
		}
		return "", err
	}
	return token.Spec.Token, nil
}

func (e *etcdKV) Exist(key string) (bool, error) {
	ctx := context.TODO()
	_, err := e.reader.GetTokenEx(ctx, key, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, err
}

func (e *etcdKV) Remove(key string) error {
	ctx := context.TODO()
	return e.writer.DeleteToken(ctx, key)
}

func (e *etcdKV) Expire(key string, expire time.Duration) error {
	ctx := context.TODO()
	token, err := e.reader.GetTokenEx(ctx, key, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return ErrNotExists
		}
		return err
	}
	token.Spec.TTL = newTTL(expire)
	_, err = e.writer.UpdateToken(ctx, token)
	return err
}

func newTTL(duration time.Duration) *int64 {
	t := int64(duration.Seconds())
	return &t
}
