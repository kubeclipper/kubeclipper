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

package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

const (
	passwordGrantType     = "password"
	refreshTokenGrantType = "refresh_token"
	verificationCodeType  = "mfa"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type handler struct {
	iamOperator           iam.Operator
	tokenOperator         auth.TokenManagementInterface
	passwordAuthenticator auth.PasswordAuthenticator
	oauth2Authenticator   auth.OAuthAuthenticator
	mfaAuthenticator      auth.MFAAuthenticator
}

func newHandler(operator iam.Operator, tokenOperator auth.TokenManagementInterface,
	passwordAuthenticator auth.PasswordAuthenticator, oauth2Authenticator auth.OAuthAuthenticator,
	mfaAuthenticator auth.MFAAuthenticator) *handler {
	return &handler{
		iamOperator:           operator,
		tokenOperator:         tokenOperator,
		passwordAuthenticator: passwordAuthenticator,
		oauth2Authenticator:   oauth2Authenticator,
		mfaAuthenticator:      mfaAuthenticator,
	}
}

func (h *handler) Token(req *restful.Request, response *restful.Response) {
	grantType, err := req.BodyParameter("grant_type")
	if err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	switch grantType {
	case passwordGrantType:
		username, _ := req.BodyParameter("username")
		password, _ := req.BodyParameter("password")
		h.passwordGrant(username, password, req, response)
	case refreshTokenGrantType:
		h.refreshTokenGrant(req, response)
	case verificationCodeType:
		h.verificationCodeGrant(req, response)
	default:
		restplus.HandleBadRequest(response, req, fmt.Errorf("grant type %s is not supported", grantType))
	}
}

func (h *handler) Login(req *restful.Request, response *restful.Response) {
	var loginRequest LoginRequest
	if err := req.ReadEntity(&loginRequest); err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	h.passwordGrant(loginRequest.Username, loginRequest.Password, req, response)
}

func (h *handler) Logout(req *restful.Request, response *restful.Response) {
	authenticated, ok := request.UserFrom(req.Request.Context())
	if ok {
		if err := h.tokenOperator.RevokeAllUserTokens(authenticated.GetName()); err != nil {
			restplus.HandleInternalError(response, req, err)
			return
		}
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) SendVerificationCode(req *restful.Request, response *restful.Response) {
	var sendSmsRequest mfa.UserMFAProvider
	if err := req.ReadEntity(&sendSmsRequest); err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	if err := h.mfaAuthenticator.ProviderRequest(sendSmsRequest); err != nil {
		restplus.HandleInternalError(response, req, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) passwordGrant(username string, password string, req *restful.Request, response *restful.Response) {
	authenticated, provider, err := h.passwordAuthenticator.Authenticate(username, password)
	if err != nil {
		formatErr := fmt.Errorf("incorrect username or password")
		rateLimitErr := fmt.Errorf("the account has been locked because of excessive password errors. please try again after 30 minutes or contact the administrator to reset your password")
		switch err {
		case auth.ErrUserOrPasswordNotValid:
			restplus.HandleUnauthorized(response, req, formatErr)
			return
		case auth.ErrUserNotExist:
			restplus.HandleUnauthorized(response, req, formatErr)
			return
		case auth.ErrIncorrectPassword:
			go h.recordLogin(username, iamv1.TokenLogin, provider, netutil.GetRequestIP(req.Request), req.Request.UserAgent(), err)
			restplus.HandleUnauthorized(response, req, formatErr)
			return
		case auth.ErrRateLimitExceeded:
			restplus.HandleTooManyRequests(response, req, rateLimitErr)
			return
		case auth.ErrAccountIsNotActive:
			restplus.HandleForbidden(response, req, err)
			return
		default:
			restplus.HandleInternalError(response, req, err)
			return
		}
	}
	if h.mfaAuthenticator.Enabled() {
		result, err := h.mfaAuthenticator.Providers(authenticated)
		if err != nil {
			restplus.HandleInternalError(response, req, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusPreconditionRequired, result)
		return
	}

	result, err := h.tokenOperator.IssueTo(authenticated)
	if err != nil {
		restplus.HandleInternalError(response, req, err)
		return
	}

	logger.Debug("user auth successful", zap.String("username", username),
		zap.String("provider", provider), zap.Strings("user_groups", authenticated.GetGroups()))

	go h.recordLogin(username, iamv1.TokenLogin, provider, netutil.GetRequestIP(req.Request), req.Request.UserAgent(), nil)
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) refreshTokenGrant(req *restful.Request, response *restful.Response) {
	refreshToken, err := req.BodyParameter("refresh_token")
	if err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	authenticated, err := h.tokenOperator.Verify(refreshToken)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}
	result, err := h.tokenOperator.IssueTo(authenticated)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) verificationCodeGrant(req *restful.Request, response *restful.Response) {
	provider, err := req.BodyParameter("mfa_provider")
	if err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	token, err := req.BodyParameter("token")
	if err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	code, err := req.BodyParameter("code")
	if err != nil {
		restplus.HandleBadRequest(response, req, err)
		return
	}
	var values url.Values
	values.Set("code", code)
	authenticated, err := h.mfaAuthenticator.Authenticate(provider, token, values)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}
	result, err := h.tokenOperator.IssueTo(authenticated)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) recordLogin(username string, loginType iamv1.LoginType, provider, sourceIP, userAgent string, authErr error) {
	// TODO: limit login record entries, the username parameter is name or email
	loginEntry := &iamv1.LoginRecord{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", username),
			Labels: map[string]string{
				common.LabelUserReference: username,
			},
		},
		Spec: iamv1.LoginRecordSpec{
			Type:      loginType,
			Provider:  provider,
			Success:   true,
			Reason:    iamv1.AuthenticatedSuccessfully,
			SourceIP:  sourceIP,
			UserAgent: userAgent,
		},
	}
	if authErr != nil {
		loginEntry.Spec.Success = false
		loginEntry.Spec.Reason = authErr.Error()
	}
	ctx := context.TODO()
	for i := 0; i < 5; i++ {
		if _, err := h.iamOperator.CreateLoginRecord(ctx, loginEntry); err != nil {
			logger.Error("create login record failed", zap.String("user", username), zap.Error(err))
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
}

func (h *handler) callback(req *restful.Request, response *restful.Response) {
	provider := req.PathParameter("callback")
	authenticated, _, err := h.oauth2Authenticator.Authenticate(provider, req.Request)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}

	result, err := h.tokenOperator.IssueTo(authenticated)
	if err != nil {
		restplus.HandleUnauthorized(response, req, err)
		return
	}
	go h.recordLogin(authenticated.GetName(), iamv1.OAuthLogin, provider, netutil.GetRequestIP(req.Request), req.Request.UserAgent(), nil)
	_ = response.WriteEntity(result)
}
