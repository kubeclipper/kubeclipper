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
	"net/http"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
)

const (
	AuthenticationTag = "Authentication"
)

func AddToContainer(c *restful.Container, operator iam.Operator, tokenOperator auth.TokenManagementInterface,
	passwordAuthenticator auth.PasswordAuthenticator, oauth2Authenticator auth.OAuthAuthenticator,
	mfaAuthenticator auth.MFAAuthenticator, authOptions *options.AuthenticationOptions, cache cache.Interface) error {
	ws := &restful.WebService{}
	ws.Path("/oauth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	h := newHandler(operator, tokenOperator, passwordAuthenticator, oauth2Authenticator, mfaAuthenticator, authOptions, cache)

	ws.Route(ws.POST("/token").
		Consumes("application/x-www-form-urlencoded").
		Doc("Oauth login").
		Notes("The resource owner password credentials grant type is suitable in\n"+
			"cases where the resource owner has a trust relationship with the\n"+
			"client, such as the device operating system or a highly privileged application.").
		Param(ws.FormParameter("grant_type", "Value MUST be set to \"password\" or \"verification_code\".").Required(true)).
		Param(ws.FormParameter("username", "The resource owner username.").Required(false)).
		Param(ws.FormParameter("password", "The resource owner password.").Required(false)).
		Param(ws.FormParameter("mfa_provider", "mfa provider type, used when grant_type is mfa").Required(false)).
		Param(ws.FormParameter("token", "mfa verification token").Required(false)).
		Param(ws.FormParameter("code", "mfa verification code").Required(false)).
		To(h.Token).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), &oauth.Token{}).
		Returns(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), errors.HTTPError{}).
		Returns(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{AuthenticationTag}))

	ws.Route(ws.GET("/cb/{callback}").
		Doc("OAuth callback API, the path param callback is config by identity provider").
		Param(ws.PathParameter("callback", "The oatuh2 idp name")).
		Param(ws.QueryParameter("access_token", "The access token issued by the authorization server.").
			Required(true)).
		Param(ws.QueryParameter("token_type", "The type of the token issued as described in [RFC6479] Section 7.1. "+
			"Value is case insensitive.").Required(true)).
		Param(ws.QueryParameter("expires_in", "The lifetime in seconds of the access token.  For "+
			"example, the value \"3600\" denotes that the access token will "+
			"expire in one hour from the time the response was generated."+
			"If omitted, the authorization server SHOULD provide the "+
			"expiration time via other means or document the default value.")).
		Param(ws.QueryParameter("scope", "if identical to the scope requested by the client;"+
			"otherwise, REQUIRED.  The scope of the access token as described by [RFC6479] Section 3.3.").Required(false)).
		Param(ws.QueryParameter("state", "if the \"state\" parameter was present in the client authorization request."+
			"The exact value received from the client.").Required(true)).
		To(h.callback).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), oauth.Token{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{AuthenticationTag}))

	ws.Route(ws.POST("/login").
		Doc("Login").
		To(h.Login).
		Reads(LoginRequest{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), &oauth.Token{}).
		Returns(http.StatusPreconditionRequired, http.StatusText(http.StatusPreconditionRequired), auth.UserMFAProviders{}).
		Returns(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), errors.HTTPError{}).
		Returns(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{AuthenticationTag}))

	ws.Route(ws.POST("/verification-code").
		Doc("send verification code").
		To(h.SendVerificationCode).
		Reads(mfa.UserMFAProvider{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), "").
		Returns(http.StatusNotFound, "session not exists", errors.HTTPError{}).
		Returns(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{AuthenticationTag}))

	ws.Route(ws.POST("/logout").
		Doc("Logout").
		Notes("logout").
		To(h.Logout).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), "").
		Metadata(restfulspec.KeyOpenAPITags, []string{AuthenticationTag}))

	c.Add(ws)
	return nil
}
