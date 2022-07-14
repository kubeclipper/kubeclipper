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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/identityprovider"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"k8s.io/apiserver/pkg/authentication/user"

	authoptions "github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
)

var _ OAuthAuthenticator = (*oauthAuthenticator)(nil)

type oauthAuthenticator struct {
	iamOperator iam.Operator
	authOptions *authoptions.AuthenticationOptions
}

func NewOauthAuthenticator(operator iam.Operator, options *authoptions.AuthenticationOptions) OAuthAuthenticator {
	return &oauthAuthenticator{
		iamOperator: operator,
		authOptions: options,
	}
}

func (o *oauthAuthenticator) Authenticate(provider string, req *http.Request) (user.Info, string, error) {
	providerOptions, err := o.authOptions.OAuthOptions.IdentityProviderOptions(provider)
	if err != nil {
		return nil, "", err
	}
	if providerOptions.MappingMethod == oauth.MappingMethodMixed || providerOptions.MappingMethod == oauth.MappingMethodLookup {
		return nil, "", fmt.Errorf("provider %s not support mapping method %s yet", provider, providerOptions.MappingMethod)
	}
	idp, err := identityprovider.GetOAuthProvider(providerOptions.Name)
	if err != nil {
		return nil, "", err
	}
	authenticated, err := idp.IdentityExchange(req)
	if err != nil {
		return nil, "", err
	}
	mappedUser, err := o.findMappedUser(providerOptions.Name, authenticated.GetUserID())
	if mappedUser == nil && providerOptions.MappingMethod == oauth.MappingMethodAuto {
		mappedUser, err = o.iamOperator.CreateUser(context.TODO(), convertUser(authenticated, providerOptions.Name))
		if err != nil {
			return nil, "", err
		}
	}
	if mappedUser != nil {
		return &user.DefaultInfo{
			Name: mappedUser.GetName(),
		}, providerOptions.Name, nil
	}
	return nil, "", fmt.Errorf("user auto mapping failed, wrap err %v", err)
}

func (o *oauthAuthenticator) findMappedUser(name string, id string) (*v1.User, error) {
	users, err := o.iamOperator.ListUsers(context.TODO(), &query.Query{
		ResourceVersion:      "0",
		LabelSelector:        fmt.Sprintf("%s=%s,%s=%s", common.LabelIDP, name, common.LabelOriginUID, id),
		ResourceVersionMatch: query.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return nil, err
	}
	if len(users.Items) != 1 {
		return nil, fmt.Errorf("user not found")
	}
	return &users.Items[0], nil
}

func convertUser(identity identityprovider.Identity, idp string) *v1.User {
	stateActive := v1.UserActive
	return &v1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(identity.GetUsername()),
			Labels: map[string]string{
				common.LabelIDP:       idp,
				common.LabelOriginUID: identity.GetUserID(),
			},
		},
		Spec: v1.UserSpec{
			Email:       identity.GetEmail(),
			DisplayName: identity.GetUsername(),
		},
		Status: v1.UserStatus{
			State: &stateActive,
		},
	}
}
