package kc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{host: u.Host, scheme: "http", client: http.DefaultClient}
}

func TestGetOperationTaskLogUsesV2TaskEndpoint(t *testing.T) {
	var request *http.Request
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		request = r
		_ = json.NewEncoder(w).Encode(oplog.LogContentResponse{Content: "log", DeliverySize: 3, LogSize: 3})
	})
	got, err := client.GetOperationTaskLog(context.Background(), "task-a", 17)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "log" {
		t.Fatalf("content = %q", got.Content)
	}
	if request.URL.Path != operationTaskPath+"/task-a/logs" {
		t.Fatalf("path = %q", request.URL.Path)
	}
	if request.URL.Query().Get("offset") != "17" {
		t.Fatalf("offset = %q", request.URL.Query().Get("offset"))
	}
}

func TestListOperationTasksFiltersByOperationUID(t *testing.T) {
	var request *http.Request
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		request = r
		_ = json.NewEncoder(w).Encode(&operationsv1alpha1.OperationTaskList{})
	})
	if _, err := client.ListOperationTasks(context.Background(), "operation-uid"); err != nil {
		t.Fatal(err)
	}
	if got := request.URL.Query().Get("fieldSelector"); got != "spec.operationRef.uid=operation-uid" {
		t.Fatalf("fieldSelector = %q", got)
	}
}

func TestOperationControlUsesCASPreconditions(t *testing.T) {
	for _, subresource := range []string{"retry", "cancel"} {
		t.Run(subresource, func(t *testing.T) {
			var request *http.Request
			var body operationsv1alpha1.OperationControlRequest
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				request = r
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				_ = json.NewEncoder(w).Encode(&operationsv1alpha1.Operation{ObjectMeta: metav1.ObjectMeta{Name: "op"}})
			})
			op := &operationsv1alpha1.Operation{ObjectMeta: metav1.ObjectMeta{Name: "op", UID: types.UID("uid"), ResourceVersion: "42"}}
			var err error
			if subresource == "retry" {
				_, err = client.RetryOperation(context.Background(), op)
			} else {
				_, err = client.CancelOperation(context.Background(), op)
			}
			if err != nil {
				t.Fatal(err)
			}
			if request.Method != http.MethodPost || request.URL.Path != operationPath+"/op/"+subresource {
				t.Fatalf("request = %s %s", request.Method, request.URL.Path)
			}
			if body.UID != "uid" || body.ResourceVersion != "42" {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}
