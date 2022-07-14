/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const (
	DefaultIssuerName = "kubeclipper"
)

type Claims struct {
	Username  string              `json:"username"`
	Groups    []string            `json:"groups,omitempty"`
	Extra     map[string][]string `json:"extra,omitempty"`
	TokenType TokenType           `json:"token_type"`
	jwt.RegisteredClaims
}

type jwtTokenIssuer struct {
	name   string
	secret []byte
	// Maximum time difference
	maximumClockSkew time.Duration
}

func (s *jwtTokenIssuer) Verify(tokenString string) (user.Info, TokenType, error) {
	clm := &Claims{}
	// verify token signature and expiration time
	_, err := jwt.ParseWithClaims(tokenString, clm, s.keyFunc)
	if err != nil {
		logger.Error("parse token failed", zap.Error(err))
		return nil, "", err
	}
	return &user.DefaultInfo{Name: clm.Username, Groups: clm.Groups, Extra: clm.Extra}, clm.TokenType, nil
}

func (s *jwtTokenIssuer) IssueTo(user user.Info, tokenType TokenType, expiresIn time.Duration) (string, error) {
	issueAt := time.Now().Add(-s.maximumClockSkew)
	notBefore := issueAt
	clm := &Claims{
		Username:  user.GetName(),
		Groups:    user.GetGroups(),
		Extra:     user.GetExtra(),
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issueAt),
			Issuer:    s.name,
			NotBefore: jwt.NewNumericDate(notBefore),
		},
	}

	if expiresIn > 0 {
		clm.ExpiresAt = jwt.NewNumericDate(clm.IssuedAt.Add(expiresIn))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, clm)

	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		logger.Error("sign token failed", zap.Error(err))
		return "", err
	}

	return tokenString, nil
}

func (s *jwtTokenIssuer) keyFunc(token *jwt.Token) (i interface{}, err error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
		return s.secret, nil
	}
	return nil, fmt.Errorf("expect token signed with HMAC but got %v", token.Header["alg"])
}

func NewTokenIssuer(secret string, maximumClockSkew time.Duration) Issuer {
	return &jwtTokenIssuer{
		name:             DefaultIssuerName,
		secret:           []byte(secret),
		maximumClockSkew: maximumClockSkew,
	}
}
