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

package generators

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

var (
	KubernetesVersion = flag.String("kc-version", "v1_0", "Version of KubeClipper to generate docs for.")
	GenKcctlDir       = flag.String("gen-kcctl-dir", "tools/kcctldocs-gen/generators", "Directory containing kcctl files")
)

const JSONOutputFile = "manifest.json"

func getTocFile() string {
	return filepath.Join(*GenKcctlDir, *KubernetesVersion, "toc.yaml")
}

func getStaticIncludesDir() string {
	return filepath.Join(*GenKcctlDir, *KubernetesVersion, "static_includes")
}

func GenerateFiles() {
	/*
		1. get command spec from cobra,about cmd description and examples.
		2. transform to markdown format.
		3. generate md file.
	*/
	pwd, err := os.Getwd()
	if err != nil {
		return
	}
	fmt.Printf("pwd:%s\n", pwd)
	// get cobra commands
	spec := GetSpec()
	tocFile := getTocFile()
	if len(tocFile) < 1 {
		fmt.Printf("Must specify --toc-file.\n")
		return
	}
	contents, err := os.ReadFile(tocFile)
	if err != nil {
		fmt.Printf("Failed to read yaml file %s: %v", getTocFile(), err)
		return
	}
	var toc ToC
	err = yaml.Unmarshal(contents, &toc)
	if err != nil {
		fmt.Printf("Unmarshal toc err:%v\n", err)
		return
	}

	NormalizeSpec(&spec)

	manifest := &Manifest{}
	manifest.Title = "Kcctl Reference Docs"
	manifest.Copyright = "<a href=\"https://github.com/kubeclipper/kubeclipper\">Copyright 2021 The kubeclipper Authors.</a>"

	staticIncludes := *GenKcctlDir + "/includes"
	if _, err = os.Stat(staticIncludes); os.IsNotExist(err) {
		if err = os.Mkdir(staticIncludes, os.FileMode(0700)); err != nil {
			fmt.Printf("Failed to create static includes directory: %v", err)
			return
		}
	}

	if err = WriteCommandFiles(manifest, toc, spec); err != nil {
		fmt.Printf("failed to write command files: %v", err)
		return
	}

	if err = WriteManifest(manifest); err != nil {
		fmt.Printf("failed to write manifest: %v", err)
		return
	}
}

func NormalizeSpec(spec *KcctlSpec) {
	for _, g := range spec.TopLevelCommandGroups {
		for _, c := range g.Commands {
			FormatCommand(c.MainCommand)
			for _, s := range c.SubCommands {
				FormatCommand(s)
			}
		}
	}
}

func FormatCommand(c *Command) {
	c.Example = FormatExample(c.Example)
	c.Description = FormatDescription(c.Description)
}

func FormatDescription(input string) string {
	/* This fixes an error when the description is a string followed by a
	   new line and another string that is indented >= four spaces. The marked.js parser
	   throws a parsing error. Error found in generated file: build/_generated_rollout.md */
	input = strings.Replace(input, "\n   ", "\n ", 10)
	return strings.Replace(input, "   *", "*", 10000)
}

func FormatExample(input string) string {
	last := ""
	result := ""
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 1 {
			continue
		}

		// Skip empty lines
		if strings.HasPrefix(line, "#") {
			if len(strings.TrimSpace(strings.Replace(line, "#", ">bdocs-tab:example", 1))) < 1 {
				continue
			}
		}

		// Format comments as code blocks
		if strings.HasPrefix(line, "#") {
			if last == "command" {
				// Close command if it is open
				result += "\n```\n\n"
			}

			if last == "comment" {
				// Add to the previous code block
				result += " " + line
			} else {
				// Start a new code block
				result += strings.Replace(line, "#", ">bdocs-tab:example", 1)
			}
			last = "comment"
		} else {
			if last != "command" {
				// Open a new code section
				result += "\n\n```bdocs-tab:example_shell"
			}
			result += "\n" + line
			last = "command"
		}
	}

	// Close the final command if needed
	if last == "command" {
		result += "\n```\n"
	}
	return result
}

func WriteManifest(manifest *Manifest) error {
	jsonBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("could not Marshal manfiest %+v due to error: %v", manifest, err)
	}
	jsonfile, err := os.Create(*GenKcctlDir + "/" + JSONOutputFile)
	if err != nil {
		return fmt.Errorf("could not create file %s due to error: %v", JSONOutputFile, err)
	}
	defer jsonfile.Close()

	_, err = jsonfile.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to write bytes %s to file %s: %v", jsonBytes, JSONOutputFile, err)
	}
	return nil
}

func WriteCommandFiles(manifest *Manifest, toc ToC, params KcctlSpec) error {
	t, err := template.New("command.template").Parse(CommandTemplate)
	if err != nil {
		return errors.WithMessage(err, "parse command.template")
	}

	m := map[string]TopLevelCommand{}
	for _, g := range params.TopLevelCommandGroups {
		for _, tlc := range g.Commands {
			m[tlc.MainCommand.Name] = tlc
		}
	}
	staticIncludesDir := getStaticIncludesDir()
	err = filepath.Walk(staticIncludesDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			to := filepath.Join(*GenKcctlDir, "includes", filepath.Base(path))
			return os.Link(path, to)
		}
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "create os.link")
	}

	for _, c := range toc.Categories {
		if len(c.Include) > 0 {
			// Use the static category include
			manifest.Docs = append(manifest.Docs, Doc{strings.ToLower(c.Include)})
		} else {
			// Write a general category include
			fn := strings.Replace(c.Name, " ", "_", -1)
			manifest.Docs = append(manifest.Docs, Doc{strings.ToLower(fmt.Sprintf("_generated_category_%s.md", fn))})
			if err = WriteCategoryFile(c); err != nil {
				return errors.WithMessage(err, "write category file")
			}
		}

		// Write each of the commands in this category
		for _, cm := range c.Commands {
			tlc, found := m[cm]
			if !found {
				return fmt.Errorf("could not find top level command %s", cm)
			}
			if err = WriteCommandFile(manifest, t, tlc); err != nil {
				return errors.WithMessage(err, "write command file")
			}
			delete(m, cm)
		}
	}
	if len(m) > 0 {
		for k := range m {
			fmt.Printf("kcctl command %s missing from table of contents", k)
		}
	}
	return nil
}

func WriteCategoryFile(c Category) error {
	ct, err := template.New("category.template").Parse(CategoryTemplate)
	if err != nil {
		return errors.WithMessage(err, "parse category.template")
	}

	fn := strings.Replace(c.Name, " ", "_", -1)
	categoryFile := *GenKcctlDir + "/includes/_generated_category_" + strings.ToLower(fmt.Sprintf("%s.md", fn))
	f, err := os.Create(categoryFile)
	if err != nil {
		return errors.WithMessage(err, "create category file")
	}
	defer f.Close()
	if err = ct.Execute(f, c); err != nil {
		return errors.WithMessage(err, "execute category template")
	}
	return nil
}

func WriteCommandFile(manifest *Manifest, t *template.Template, params TopLevelCommand) error {
	replacer := strings.NewReplacer(
		"|", "&#124;",
		"<", "&lt;",
		">", "&gt;",
		"[", "<span>[</span>",
		"]", "<span>]</span>",
		"\n", "<br>",
	)

	params.MainCommand.Description = replacer.Replace(params.MainCommand.Description)
	for _, o := range params.MainCommand.Options {
		o.Usage = replacer.Replace(o.Usage)
	}
	for _, sc := range params.SubCommands {
		for _, o := range sc.Options {
			o.Usage = replacer.Replace(o.Usage)
		}
	}
	generatedFile := *GenKcctlDir + "/includes/_generated_" + strings.ToLower(params.MainCommand.Name) + ".md"
	f, err := os.Create(generatedFile)
	if err != nil {
		fmt.Printf("Failed to open index: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	err = t.Execute(f, params)
	if err != nil {
		return errors.WithMessage(err, "execute template")
	}
	manifest.Docs = append(manifest.Docs, Doc{"_generated_" + strings.ToLower(params.MainCommand.Name) + ".md"})
	return nil
}
