/*
 *
 *  * Copyright 2026 KubeClipper Authors.
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

package v1alpha1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"
	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestPublicInterfacesMatchRegisteredRoutes(t *testing.T) {
	t.Parallel()

	operationInterface := reflect.TypeFor[OperationInterface]()
	for _, method := range []string{"Update", "UpdateStatus", "Delete", "DeleteCollection", "Patch"} {
		if _, found := operationInterface.MethodByName(method); found {
			t.Errorf("OperationInterface exposes unregistered method %s", method)
		}
	}

	taskInterface := reflect.TypeFor[OperationTaskInterface]()
	for _, method := range []string{"Create", "Update", "Delete", "DeleteCollection", "Patch"} {
		if _, found := taskInterface.MethodByName(method); found {
			t.Errorf("OperationTaskInterface exposes unregistered method %s", method)
		}
	}
	if _, found := taskInterface.MethodByName("UpdateStatus"); !found {
		t.Error("OperationTaskInterface must expose the registered status subresource")
	}
}

func TestOperationTaskListUsesOperationsAPIPathAndListOptions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/operations.kubeclipper.io/v1alpha1/operationtasks"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("fieldSelector"), "spec.nodeRef.name=node-a"; got != want {
			t.Errorf("fieldSelector = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("resourceVersion"), "42"; got != want {
			t.Errorf("resourceVersion = %q, want %q", got, want)
		}
		if got := r.Header.Get(clientrest.QueryTypeHeader); got != "" {
			t.Errorf("mTLS-style client unexpectedly sent %s=%q", clientrest.QueryTypeHeader, got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&operationsv1alpha1.OperationTaskList{
			TypeMeta: metav1.TypeMeta{APIVersion: operationsv1alpha1.SchemeGroupVersion.String(), Kind: operationsv1alpha1.KindOperationTask + "List"},
			ListMeta: metav1.ListMeta{ResourceVersion: "43"},
		}); err != nil {
			t.Errorf("encode task list: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig() error = %v", err)
	}
	list, err := client.OperationTasks().List(context.Background(), metav1.ListOptions{
		FieldSelector:   "spec.nodeRef.name=node-a",
		ResourceVersion: "42",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := list.ResourceVersion, "43"; got != want {
		t.Fatalf("list resourceVersion = %q, want %q", got, want)
	}
}

func TestOperationTaskUpdateStatusUsesStatusSubresource(t *testing.T) {
	t.Parallel()

	task := &operationsv1alpha1.OperationTask{
		TypeMeta: metav1.TypeMeta{APIVersion: operationsv1alpha1.SchemeGroupVersion.String(), Kind: operationsv1alpha1.KindOperationTask},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "task-a",
			UID:             types.UID("task-uid"),
			ResourceVersion: "7",
		},
		Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskRunning},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPut; got != want {
			t.Errorf("request method = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/api/operations.kubeclipper.io/v1alpha1/operationtasks/task-a/status"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}
		var request operationsv1alpha1.OperationTask
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.UID != task.UID || request.ResourceVersion != task.ResourceVersion ||
			request.Status.Phase != operationsv1alpha1.TaskRunning {
			t.Errorf("status request lost CAS identity or status: %#v", request)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(task); err != nil {
			t.Errorf("encode task: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig() error = %v", err)
	}
	if _, err := client.OperationTasks().UpdateStatus(context.Background(), task, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
}

func TestOperationControlUsesNamedSubresourceAndCASBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Errorf("request method = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/api/operations.kubeclipper.io/v1alpha1/operations/op-a/retry"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}
		var request operationsv1alpha1.OperationControlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.UID != types.UID("op-uid") || request.ResourceVersion != "9" {
			t.Errorf("control request lost CAS preconditions: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&operationsv1alpha1.Operation{
			TypeMeta:   metav1.TypeMeta{APIVersion: operationsv1alpha1.SchemeGroupVersion.String(), Kind: operationsv1alpha1.KindOperation},
			ObjectMeta: metav1.ObjectMeta{Name: "op-a", UID: types.UID("op-uid"), ResourceVersion: "10"},
		}); err != nil {
			t.Errorf("encode operation: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Operations().Retry(context.Background(), "op-a", &operationsv1alpha1.OperationControlRequest{
		UID: types.UID("op-uid"), ResourceVersion: "9",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceVersion != "10" {
		t.Fatalf("result resourceVersion = %q, want 10", result.ResourceVersion)
	}
}
