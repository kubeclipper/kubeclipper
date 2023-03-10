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
	"errors"
	"fmt"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/random"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"k8s.io/apiserver/pkg/authentication/authenticator"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"

	authoptions "github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/token"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/query"
)

type tokenOperator struct {
	issuer     token.Issuer
	options    *authoptions.AuthenticationOptions
	tokenCache iam.Operator
}

var _ TokenManagementInterface = (*tokenOperator)(nil)

func NewTokenOperator(operator iam.Operator, opts *authoptions.AuthenticationOptions) TokenManagementInterface {
	return &tokenOperator{
		issuer:     token.NewTokenIssuer(opts.JwtSecret, opts.MaximumClockSkew),
		options:    opts,
		tokenCache: operator,
	}
}

func (t *tokenOperator) IssueTo(user user.Info) (*oauth.Token, error) {
	accessTokenExpiresIn := t.options.OAuthOptions.AccessTokenMaxAge
	refreshTokenExpiresIn := accessTokenExpiresIn + t.options.OAuthOptions.AccessTokenInactivityTimeout

	accessToken, err := t.issuer.IssueTo(user, token.AccessToken, accessTokenExpiresIn)
	if err != nil {
		return nil, err
	}
	refreshToken, err := t.issuer.IssueTo(user, token.RefreshToken, refreshTokenExpiresIn)
	if err != nil {
		return nil, err
	}

	result := &oauth.Token{
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		RefreshToken:     refreshToken,
		ExpiresIn:        int(accessTokenExpiresIn.Seconds()),
		RefreshExpiresIn: int(refreshTokenExpiresIn.Seconds()),
	}

	if !t.options.MultipleLogin {
		if err := t.RevokeAllUserTokens(user.GetName()); err != nil {
			return nil, err
		}
	}

	if accessTokenExpiresIn > 0 {
		if err = t.cacheToken(user.GetName(), accessToken, iamv1.AccessToken, accessTokenExpiresIn); err != nil {
			return nil, err
		}
		if err = t.cacheToken(user.GetName(), refreshToken, iamv1.RefreshToken, refreshTokenExpiresIn); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (t *tokenOperator) RevokeAllUserTokens(username string) error {
	// TODO: make async, is too expensive to spend time for delete token
	return t.tokenCache.DeleteTokenCollection(context.TODO(), &query.Query{
		Pagination:      query.NoPagination(),
		TimeoutSeconds:  nil,
		ResourceVersion: "",
		Watch:           false,
		LabelSelector:   fmt.Sprintf("%s=%s", common.LabelUsername, username),
		FieldSelector:   "",
		Continue:        "",
		Limit:           0,
		Reverse:         false,
		OrderBy:         "",
	})
}

func (t *tokenOperator) Verify(tokenStr string) (user.Info, error) {
	authenticated, tokenType, err := t.issuer.Verify(tokenStr)
	if err != nil {
		return nil, err
	}
	if t.options.OAuthOptions.AccessTokenMaxAge == 0 ||
		tokenType == token.StaticToken {
		return authenticated, nil
	}
	if err = t.tokenCacheValidate(authenticated.GetName(), tokenStr); err != nil {
		return nil, err
	}
	return authenticated, nil
}

func (t *tokenOperator) cacheToken(username, token string, tokenType iamv1.TokenType, duration time.Duration) error {
	_, err := t.tokenCache.CreateToken(context.TODO(), &iamv1.Token{
		TypeMeta: metav1.TypeMeta{
			Kind:       iamv1.KindToken,
			APIVersion: iamv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				common.LabelUsername: username,
			},
			Name: fmt.Sprintf("%s-%s", username, random.GenerateString(6)),
		},
		Spec: iamv1.TokenSpec{
			TokenType: tokenType,
			Token:     token,
			TTL:       newTTL(duration),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (t *tokenOperator) tokenCacheValidate(username, tokenStr string) error {
	tokens, err := t.tokenCache.ListTokens(context.TODO(), &query.Query{
		Pagination:      query.NoPagination(),
		ResourceVersion: "0",
		LabelSelector:   fmt.Sprintf("%s=%s", common.LabelUsername, username),
		FieldSelector:   fmt.Sprintf("spec.token=%s", tokenStr),
	})
	if err != nil {
		return err
	}
	if len(tokens.Items) > 0 {
		return nil
	}
	return errors.New("token invalid in cache")
}

func newTTL(duration time.Duration) *int64 {
	t := int64(duration.Seconds())
	return &t
}

type tokenAuthenticator struct {
	userOperator  iam.Operator
	tokenOperator TokenManagementInterface
}

func NewTokenAuthenticator(operator iam.Operator, managementInterface TokenManagementInterface) authenticator.Token {
	return &tokenAuthenticator{
		userOperator:  operator,
		tokenOperator: managementInterface,
	}
}

func (t *tokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {
	providedUser, err := t.tokenOperator.Verify(token)
	if err != nil {
		return nil, false, err
	}

	// TODO: oatuh user

	dbUser, err := t.userOperator.GetUserEx(ctx, providedUser.GetName(), "0", false, false)
	if err != nil {
		return nil, false, err
	}

	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   dbUser.GetName(),
			Groups: append(dbUser.Spec.Groups, user.AllAuthenticated),
		},
	}, true, nil
}
