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

package client

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

func Push(file, registryStr string) error {
	manifest, err := tarball.LoadManifest(pathOpener(file))
	if err != nil {
		return errors.WithMessage(err, "load manifest")
	}
	for i, descriptor := range manifest {
		for _, t := range descriptor.RepoTags {
			tag, err := name.NewTag(t)
			if err != nil {
				return errors.WithMessage(err, "new tag")
			}
			img, err := tarball.Image(pathOpener(file), &tag)
			if err != nil {
				return errors.WithMessage(err, "image")
			}
			// push to specify registry
			repository := tag.RepositoryStr()
			target := fmt.Sprintf("%s/%s:%s", registryStr, repository, tag.TagStr())
			logger.V(2).Infof("[%v/%v] push %s -> %s\n", i+1, len(manifest), t, target)
			if err = crane.Push(img, target, crane.Insecure); err != nil {
				return errors.WithMessage(err, "push")
			}

			// if image without organization，we push twice。
			// e.g. nginx:1.20,once is library/nginx:1.20,once is nginx:1.20.
			// NOTE：because ks specify library in yaml,so we must add library,
			// but some image doesn't specify library in yaml,so we must push without library
			// for compatible，we push twice.
			split := strings.Split(repository, "/")
			if len(split) == 2 && split[0] == "library" {
				repository = split[1]
				target = fmt.Sprintf("%s/%s:%s", registryStr, repository, tag.TagStr())
				logger.V(2).Infof("[%v/%v] push %s -> %s\n", i+1, len(manifest), t, target)
				if err = crane.Push(img, target, crane.Insecure); err != nil {
					return errors.WithMessage(err, "push")
				}
			}
		}
	}
	return nil
}

// Delete docker registry doesn't support tag deletion and we have to delete by digest
// e.g.  crane delete --insecure "localhost:5000/caas4/nfsplugin@$(crane digest localhost:5000/caas4/nfsplugin:v4.1.0)"
func Delete(registry, repository, tag string) error {
	digest, err := crane.Digest(fmt.Sprintf("%s/%s:%s", registry, repository, tag), crane.Insecure)
	if err != nil {
		return errors.WithMessage(err, "get digest")
	}
	return crane.Delete(fmt.Sprintf("%s/%s@%s", registry, repository, digest), crane.Insecure)
}

// Catalog return repositories in registry.
func Catalog(registry string) ([]string, error) {
	return crane.Catalog(registry, crane.Insecure)
}

// ListTags return repositories in registry.
func ListTags(registry, repository string) ([]string, error) {
	return crane.ListTags(fmt.Sprintf("%s/%s", registry, repository), crane.Insecure)
}

func pathOpener(path string) tarball.Opener {
	return func() (io.ReadCloser, error) {
		return os.Open(path)
	}
}
