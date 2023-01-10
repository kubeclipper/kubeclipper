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

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	auditingv1 "github.com/kubeclipper/kubeclipper/pkg/apis/auditing/v1"
	"github.com/kubeclipper/kubeclipper/pkg/apis/proxy"

	"github.com/kubeclipper/kubeclipper/pkg/apis/oauth"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/apis/iam/v1"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	urlruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/version"

	configv1 "github.com/kubeclipper/kubeclipper/pkg/apis/config/v1"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

var output string

func init() {
	flag.StringVar(&output, "output", "./api/openapi-spec/swagger.json", "--output=./api.json")
}

func main() {
	flag.Parse()
	swaggerSpec := generateSwaggerJSON()

	err := validateSpec(swaggerSpec)
	if err != nil {
		logger.Warn("Swagger specification has errors")
	}
}

func validateSpec(apiSpec []byte) error {

	swaggerDoc, err := loads.Analyzed(apiSpec, "2.0")
	if err != nil {
		return err
	}

	// Attempts to report about all errors
	validate.SetContinueOnErrors(true)

	v := validate.NewSpecValidator(swaggerDoc.Schema(), strfmt.Default)
	result, _ := v.Validate(swaggerDoc)

	if result.HasWarnings() {
		log.Printf("See warnings below:\n")
		for _, desc := range result.Warnings {
			log.Printf("- WARNING: %s\n", desc.Error())
		}

	}
	if result.HasErrors() {
		str := fmt.Sprintf("The swagger spec is invalid against swagger specification %s.\nSee errors below:\n", swaggerDoc.Version())
		for _, desc := range result.Errors {
			str += fmt.Sprintf("- %s\n", desc.Error())
		}
		log.Println(str)
		return errors.New(str)
	}

	return nil
}

func generateSwaggerJSON() []byte {

	container := restful.NewContainer()
	urlruntime.Must(corev1.AddToContainer(container, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	urlruntime.Must(iamv1.AddToContainer(container, nil, nil, nil))
	urlruntime.Must(configv1.AddToContainer(container, nil, nil))
	urlruntime.Must(oauth.AddToContainer(container, nil, nil, nil, nil, nil, nil, nil))
	urlruntime.Must(proxy.AddToContainer(container, nil))
	urlruntime.Must(auditingv1.AddToContainer(container, nil))

	config := restfulspec.Config{
		WebServices:                   container.RegisteredWebServices(),
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}

	swagger := restfulspec.BuildSwagger(config)
	swagger.Info.Extensions = make(spec.Extensions)
	swagger.Info.Extensions.Add("x-tagGroups", []struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{
		// {
		//	Name: "Cluster",
		//	Tags: []string{
		//		constants.CoreClusterTag,
		//	},
		// },
	})

	data, _ := json.MarshalIndent(swagger, "", "  ")
	err := os.WriteFile(output, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("successfully written to %s", output)

	return data
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeClipper",
			Description: "KubeClipper OpenAPI",
			Version:     version.Get().GitVersion,
			Contact: &spec.ContactInfo{
				// TODO: add url and email
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "KubeClipper",
					URL:   "https://github.com/kubeclipper-labs/kubeclipper",
					Email: "",
				},
			},
			License: &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: "Apache 2.0",
					URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
				},
			},
		},
	}

	// setup security definitions
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"jwt": spec.APIKeyAuth("Authorization", "header"),
	}
	swo.Security = []map[string][]string{{"jwt": []string{}}}
}

// func apiTree(container *restful.Container) {
//	buf := bytes.NewBufferString("\n")
//	for _, ws := range container.RegisteredWebServices() {
//		for _, route := range ws.Routes() {
//			buf.WriteString(fmt.Sprintf("%s %s\n", route.Method, route.Path))
//		}
//	}
//	log.Println(buf.String())
// }
