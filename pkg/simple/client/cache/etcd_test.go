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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	iammock "github.com/kubeclipper/kubeclipper/pkg/models/iam/mock"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type tokenMatchFunc func(*iamv1.Token) bool

func (m tokenMatchFunc) Matches(x interface{}) bool {
	token := x.(*iamv1.Token)
	return m(token)
}

func (m tokenMatchFunc) String() string {
	return "func matcher"
}

func initMockToken(t *testing.T, ctrl *gomock.Controller) (iam.TokenReader, iam.TokenWriter) {
	asserts := assert.New(t)
	mock := iammock.NewMockOperator(ctrl)

	// for exist
	mock.EXPECT().GetTokenEx(gomock.Any(), "foo", "0").
		Return(nil, apimachineryErrors.NewNotFound(schema.GroupResource{}, "")).
		Times(1)

	var expToken iamv1.Token

	// for create
	mock.EXPECT().CreateToken(gomock.Any(), tokenMatchFunc(func(token *iamv1.Token) bool {
		expToken = *token.DeepCopy()
		expireAt := metav1.NewTime(time.Now().Add(time.Second * 4))
		expToken.Status.ExpiresAt = &expireAt
		return asserts.Equal("foo", token.Name) &&
			asserts.Equal(iamv1.TemporaryToken, token.Spec.TokenType) &&
			asserts.Equal("bar", token.Spec.Token) &&
			asserts.Equal(newTTL(time.Second*5), token.Spec.TTL)
	})).Return(&expToken, nil).Times(1)

	// for exist,get,expire,expire after get
	mock.EXPECT().GetTokenEx(gomock.Any(), "foo", "0").Return(&expToken, nil).Times(4)

	// for expire
	mock.EXPECT().UpdateToken(gomock.Any(), tokenMatchFunc(func(token *iamv1.Token) bool {
		expToken = *token.DeepCopy()
		expireAt := metav1.NewTime(time.Now().Add(time.Second * 4))
		expToken.Status.ExpiresAt = &expireAt
		return asserts.Equal("foo", token.Name) &&
			asserts.Equal(iamv1.TemporaryToken, token.Spec.TokenType) &&
			asserts.Equal("bar", token.Spec.Token) &&
			asserts.Equal(newTTL(time.Second*10), token.Spec.TTL)
	})).Return(&expToken, nil).Times(1)

	// for delete
	mock.EXPECT().DeleteToken(gomock.Any(), gomock.Eq("foo")).Return(nil).Times(1)

	// for delete after get
	mock.EXPECT().GetTokenEx(gomock.Any(), "foo", "0").
		Return(nil, apimachineryErrors.NewNotFound(schema.GroupResource{}, "")).
		Times(1)

	return mock, mock
}

func TestEtcdKVStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kv, err := NewEtcd(initMockToken(t, ctrl))
	require.NoError(t, err)
	CacheCommonTest(t, kv)

}
