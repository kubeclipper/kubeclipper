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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadAuthenticationJWTSecret(t *testing.T) {
	for _, tt := range []struct {
		name    string
		config  string
		wantErr bool
	}{
		{name: "authentication omitted", config: "generic: {}\n", wantErr: true},
		{name: "secret omitted", config: "authentication: {}\n", wantErr: true},
		{name: "empty secret", config: "authentication:\n  jwtSecret: \"\"\n", wantErr: true},
		{name: "public default", config: "authentication:\n  jwtSecret: kubeclipper\n", wantErr: true},
		{name: "configured", config: "authentication:\n  jwtSecret: D9ykGOmuE3yXe35Wh3mRniGT\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			path := filepath.Join(t.TempDir(), "kubeclipper-server.yaml")
			if err := os.WriteFile(path, []byte(tt.config), 0600); err != nil {
				t.Fatal(err)
			}
			viper.AddConfigPath(filepath.Dir(path))
			conf, err := TryLoadFromDisk()
			if err != nil {
				t.Fatal(err)
			}
			errs := conf.AuthenticationOptions.Validate()
			if (len(errs) > 0) != tt.wantErr {
				t.Fatalf("Validate() errors = %v, want error = %v", errs, tt.wantErr)
			}
		})
	}
}
