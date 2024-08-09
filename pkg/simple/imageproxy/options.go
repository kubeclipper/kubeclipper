/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package imageproxy

import (
	"fmt"
	"net/url"

	"github.com/spf13/pflag"
)

type Options struct {
	KcImageRepoMirror string `json:"kcImageRepoMirror" yaml:"kcImageRepoMirror"`
}

func NewOptions() *Options {
	return &Options{
		KcImageRepoMirror: "",
	}
}

func (s *Options) Validate() []error {
	var errs []error
	uri, err := url.Parse(s.KcImageRepoMirror)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	if uri.Scheme != "" {
		errs = append(errs, fmt.Errorf("kc image repo mirror is not support incoming protcol: %s", uri.Scheme))
	}
	return errs
}

func (s *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.KcImageRepoMirror, "kc-image-repo-mirror", s.KcImageRepoMirror, "K8s image repository mirror")
}
