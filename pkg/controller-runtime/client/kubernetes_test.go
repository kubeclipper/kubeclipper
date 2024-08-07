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
	"encoding/base64"
	"testing"
)

func TestFromKubeConfig(t *testing.T) {
	kubeconfigBase64 := "CmFwaVZlcnNpb246IHYxCmNsdXN0ZXJzOgotIGNsdXN0ZXI6CiAgICBpbnNlY3VyZS1za2lwLXRscy12ZXJpZnk6IHRydWUKICAgIHNlcnZlcjogaHR0cHM6Ly8xOTIuMTY4LjEwLjIwOTo2NDQzCiAgbmFtZToga2MKY29udGV4dHM6Ci0gY29udGV4dDoKICAgIGNsdXN0ZXI6IGtjCiAgICB1c2VyOiBrYy1zZXJ2ZXIKICBuYW1lOiBrYy1zZXJ2ZXIta2MKY3VycmVudC1jb250ZXh0OiBrYy1zZXJ2ZXIta2MKa2luZDogQ29uZmlnCnByZWZlcmVuY2VzOiB7fQp1c2VyczoKLSBuYW1lOiBrYy1zZXJ2ZXIKICB1c2VyOgogICAgdG9rZW46IGV5SmhiR2NpT2lKU1V6STFOaUlzSW10cFpDSTZJak5MUVhOVGNYcGhUemhYTkRaUFFUUldZbkp2UTFreGVVaFJWVGswUTBsS1NuTlROME5oTVRCa1FUUWlmUS5leUpwYzNNaU9pSnJkV0psY201bGRHVnpMM05sY25acFkyVmhZMk52ZFc1MElpd2lhM1ZpWlhKdVpYUmxjeTVwYnk5elpYSjJhV05sWVdOamIzVnVkQzl1WVcxbGMzQmhZMlVpT2lKcmRXSmxMWE41YzNSbGJTSXNJbXQxWW1WeWJtVjBaWE11YVc4dmMyVnlkbWxqWldGalkyOTFiblF2YzJWamNtVjBMbTVoYldVaU9pSnJZeTF6WlhKMlpYSXRkRzlyWlc0dE9HaGlaellpTENKcmRXSmxjbTVsZEdWekxtbHZMM05sY25acFkyVmhZMk52ZFc1MEwzTmxjblpwWTJVdFlXTmpiM1Z1ZEM1dVlXMWxJam9pYTJNdGMyVnlkbVZ5SWl3aWEzVmlaWEp1WlhSbGN5NXBieTl6WlhKMmFXTmxZV05qYjNWdWRDOXpaWEoyYVdObExXRmpZMjkxYm5RdWRXbGtJam9pTVRnM1ptTXdOemt0WlRSak1pMDBZak5sTFdFeU5qWXROV1l5TjJSbU56UmtaalE0SWl3aWMzVmlJam9pYzNsemRHVnRPbk5sY25acFkyVmhZMk52ZFc1ME9tdDFZbVV0YzNsemRHVnRPbXRqTFhObGNuWmxjaUo5LkV0OE9fTzh5Vzl1SXdWRFpTeDdrdWVYX19QZTVZOW9rSWQ2azc3QVRZY1pMQXdFanVzMnNwYUdhS2F6bmM0TU1rX0tCMWcyWHNzVlVDaGRJampSb2pNTTg1bVNvTG42R294NGtGSzZKOXprRWZRNXV4ZXMyLUxTNzhOcm94YWF0eVJzUWIwTlhMaXd2X0EyVkw1dXBsOFNpWWVOM3JGTGwxc0Z5cXpTNWl2VzJxZFBpSk1aVkRoZEY1M0tRa09GbURqOVdyN0tYVjJNVThYQU42R0NSa2NYWjdVcGFOLWxaZ2tfRFpLMjJTLWhWaDJHX1o1Vkg4RVR4YUpla3kwOVBfRlJBODQxODMzUTExVi0wLUtTUnQtMXdtZUk4a0hfSW95X3NDRC0zNlRNeWhLMFNKdTBOUURacjNXeU9ndVFLMEtBbVF1VExZMUhpdVo1ZktPUi1Fdw=="
	kubeConfig, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = FromKubeConfig(kubeConfig)
	if err != nil {
		t.Fatal(err)
	}
}
