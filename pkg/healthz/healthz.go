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

package healthz

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HealthChecker is a named healthz checker.
type HealthChecker interface {
	Name() string
	Check(req *http.Request) error
}

// healthzCheck implements HealthChecker on an arbitrary name and check function.
type healthzCheck struct {
	name  string
	check func(r *http.Request) error
}

var _ HealthChecker = &healthzCheck{}

func (c *healthzCheck) Name() string {
	return c.name
}

func (c *healthzCheck) Check(r *http.Request) error {
	return c.check(r)
}

// NamedCheck returns a healthz checker for the given name and function.
func NamedCheck(name string, check func(r *http.Request) error) HealthChecker {
	return &healthzCheck{name, check}
}

func InstallRootHealthz(c *restful.Container, checks ...HealthChecker) {
	c.HandleWithFilter("/healthz", handleRootHealthz())
	for _, check := range checks {
		c.HandleWithFilter(fmt.Sprintf("%s/%v", "/healthz", check.Name()), adaptCheckToHandler(check.Check))
	}
}

// getExcludedChecks extracts the health check names to be excluded from the query param
func getExcludedChecks(r *http.Request) sets.Set[string] {
	checks, found := r.URL.Query()["exclude"]
	if found {
		return sets.New(checks...)
	}
	return sets.Set[string]{}
}

// formatQuoted returns a formatted string of the health check names,
// preserving the order passed in.
func formatQuoted(names ...string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, fmt.Sprintf("%q", name))
	}
	return strings.Join(quoted, ",")
}

// handleRootHealthz returns an http.HandlerFunc that serves the provided checks.
func handleRootHealthz(checks ...HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		failed := false
		excluded := getExcludedChecks(r)
		var verboseOut bytes.Buffer
		for _, check := range checks {
			// no-op the check if we've specified we want to exclude the check
			if excluded.Has(check.Name()) {
				excluded.Delete(check.Name())
				fmt.Fprintf(&verboseOut, "[+]%v excluded: ok\n", check.Name())
				continue
			}
			if err := check.Check(r); err != nil {
				// don't include the error since this endpoint is public.  If someone wants more detail
				// they should have explicit permission to the detailed checks.
				fmt.Fprintf(&verboseOut, "[-]%v failed: reason withheld\n", check.Name())
				failed = true
			} else {
				fmt.Fprintf(&verboseOut, "[+]%v ok\n", check.Name())
			}
		}
		if excluded.Len() > 0 {
			fmt.Fprintf(&verboseOut, "warn: some health checks cannot be excluded: no matches for %v\n", formatQuoted(sets.List(excluded)...))
		}
		// always be verbose on failure
		if failed {
			http.Error(w, fmt.Sprintf("%vhealthz check failed", verboseOut.String()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if _, found := r.URL.Query()["verbose"]; !found {
			fmt.Fprint(w, "ok")
			return
		}

		_, _ = verboseOut.WriteTo(w)
		_, _ = fmt.Fprint(w, "healthz check passed\n")
	}
}

// adaptCheckToHandler returns an http.HandlerFunc that serves the provided checks.
func adaptCheckToHandler(c func(r *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := c(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		} else {
			_, _ = fmt.Fprint(w, "ok")
		}
	})
}
