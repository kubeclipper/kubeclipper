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

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/sync/errgroup"
)

const tmpl = `/*
 *
 *  * Copyright %s KubeClipper Authors.
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
`

var (
	ignorePatterns = stringSlice{"api/**", "deploy/**", "dist/**", "vendor/**", "docs/**"}
	year           = flag.String("y", fmt.Sprint(time.Now().Year()), "copyright year(s)")
	verbose        = flag.Bool("v", false, "verbose mode: print the name of the files that are modified or were skipped")
)

func init() {
	flag.Var(&ignorePatterns, "i", "file patterns to ignore, for example: -i vendor/**")
}

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprint(*i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type file struct {
	path string
	mode os.FileMode
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, p := range ignorePatterns {
		if !doublestar.ValidatePattern(p) {
			log.Fatalf("ignore pattern %q is invalid", p)
		}
	}

	headerText := fmt.Sprintf(tmpl, *year)

	// process at most CPU number files in parallel
	ch := make(chan *file, runtime.NumCPU())
	done := make(chan struct{})
	go func() {
		var wg errgroup.Group
		for f := range ch {
			f := f
			wg.Go(func() error {
				updated, err := addLicenseHeader(f.path, headerText, f.mode)
				if err != nil {
					log.Printf("%s: %v", f.path, err)
					return err
				}
				if *verbose && updated {
					log.Printf("%s updated", f.path)
				}
				return nil
			})
		}
		err := wg.Wait()
		close(done)
		if err != nil {
			os.Exit(1)
		}
	}()

	for _, d := range flag.Args() {
		if err := walk(ch, d); err != nil {
			log.Fatal(err)
		}
	}
	close(ch)
	<-done
}

// walk walks the file tree from root, sends go file to task queue.
func walk(ch chan<- *file, root string) error {
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%s error: %v", path, err)
		}
		if fi.IsDir() {
			return nil
		}
		if match(path, ignorePatterns) || filepath.Ext(path) != ".go" {
			if *verbose {
				log.Printf("skipping: %s", path)
			}
			return nil
		}
		ch <- &file{path, fi.Mode()}
		return nil
	})
}

// match returns if path matches one of the provided file patterns.
func match(path string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := doublestar.Match(p, path); ok {
			return true
		}
	}
	return false
}

// hasLicense returns if there is a license in the file header already.
func hasLicense(header []byte) bool {
	return bytes.Contains(header, []byte("Copyright")) &&
		bytes.Contains(header, []byte("Apache License"))
}

// addLicenseHeader adds a license header to the file if it does not exist.
func addLicenseHeader(path, license string, fmode os.FileMode) (bool, error) {
	f, err := os.OpenFile(path, os.O_RDWR, fmode)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf1, buf2, dropped := scanLines(f)
	if hasLicense(buf1.Bytes()) {
		return false, nil
	}

	// Create a temporary go file for injecting license headers.
	// The prefix `addlicense-` is used for batch deleting the temporary files if problems occur.
	tmp, err := os.CreateTemp(filepath.Dir(path), "addlicense-")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())

	stat, err := f.Stat()
	if err != nil {
		return false, err
	}
	// Copy the file mode.
	if err := tmp.Chmod(stat.Mode()); err != nil {
		return false, err
	}

	// if err := tmp.Chown(int(stat.Sys().(*syscall.Stat_t).Uid), int(stat.Sys().(*syscall.Stat_t).Gid)); err != nil {
	// 	return false, err
	// }

	// This go file has build constraints.
	if buf2.Len() > 0 {
		if _, err := tmp.WriteString(buf2.String() + "\n"); err != nil {
			return false, err
		}
	}

	// Write license header firstly.
	if _, err := tmp.WriteString(license + "\n"); err != nil {
		return false, err
	}

	// Caculate position of the line contains `package `.
	offset := int64(buf1.Len())
	if dropped > 1 {
		// CR characters are dropped, they should be added back.
		offset += dropped - 1
	}
	// Move the cursor of original go file to the position.
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return false, err
	}

	// Pour rest of the original go file into the temporary one.
	if _, err := io.Copy(tmp, f); err != nil {
		return false, err
	}

	if err := tmp.Sync(); err != nil {
		return false, err
	}

	// replace the original file
	err = os.Rename(tmp.Name(), f.Name())
	return err == nil, err
}

// scanLines handles the file line by line
// buffer1: the lines before the package declaration
// buffer2: the lines of go build constraints
// dropped: the number of CR (`\r`) characters dropped
func scanLines(file *os.File) (buf1, buf2 bytes.Buffer, dropped int64) {
	scanner := bufio.NewScanner(file)
	dropCR := func(data []byte) []byte {
		if len(data) > 0 && data[len(data)-1] == '\r' {
			dropped++
			return data[0 : len(data)-1]
		}
		return data
	}
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			// We have a full newline-terminated line.
			return i + 1, dropCR(data[0:i]), nil
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), dropCR(data), nil
		}
		// Request more data.
		return 0, nil, nil
	})
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.HasPrefix(line, []byte("package ")) {
			// Reading until the line contains `package p`.
			break
		} else if bytes.HasPrefix(line, []byte("//go:build ")) || bytes.HasPrefix(line, []byte("// +build ")) {
			buf2.Write(line)
			buf2.WriteString("\n")
		}
		buf1.Write(line)
		buf1.WriteString("\n")
	}
	return
}
