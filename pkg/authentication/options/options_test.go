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

package options

import (
	"crypto/rand"
	"testing"
)

func TestAuthenticationOptionsValidateJWTSecret(t *testing.T) {
	for _, tt := range []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "public default", secret: "kubeclipper", wantErr: true},
		{name: "configured", secret: rand.Text()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewAuthenticateOptions()
			if opts.JwtSecret != "" {
				t.Fatal("authentication options must not provide a default JWT secret")
			}
			opts.JwtSecret = tt.secret
			errs := opts.Validate()
			if (len(errs) > 0) != tt.wantErr {
				t.Fatalf("Validate() errors = %v, want error = %v", errs, tt.wantErr)
			}
		})
	}
}
