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

package restplus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// singleEventWatch streams one preloaded event set and then closes the
// channel, so ServeWatch returns deterministically.
type singleEventWatch struct {
	ch chan watch.Event
}

func newSingleEventWatch(events ...watch.Event) *singleEventWatch {
	ch := make(chan watch.Event, len(events)+1)
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return &singleEventWatch{ch: ch}
}

func (*singleEventWatch) Stop() {}

func (w *singleEventWatch) ResultChan() <-chan watch.Event { return w.ch }

// A 410 "too old resource version" travels the watch stream as an ERROR event
// whose payload is an *apierrors.StatusError. It must reach the client as a
// decodable metav1.Status so the reflector relists instead of failing the
// stream with "Object 'Kind' is missing".
func TestServeWatchEncodesErrorEventAsDecodableStatus(t *testing.T) {
	expired := apierrors.NewResourceExpired("too old resource version: 1, 100")
	status := expired.Status()
	watcher := newSingleEventWatch(watch.Event{Type: watch.Error, Object: &status})

	httpReq := httptest.NewRequestWithContext(context.Background(), "GET", "/apis/core.kubeclipper.io/v1/clusters?watch=true", http.NoBody)
	httpReq.Header.Set("Accept", "application/json")
	request := restful.NewRequest(httpReq)
	recorder := httptest.NewRecorder()
	response := restful.NewResponse(recorder)

	ServeWatch(watcher, corev1.SchemeGroupVersion.WithKind("Cluster"), request, response, 10*time.Second)

	frameReader := json.Framer.NewFrameReader(io.NopCloser(bytes.NewReader(recorder.Body.Bytes())))
	streamDecoder := streaming.NewDecoder(frameReader, scheme.Codecs.UniversalDeserializer())

	var sawErrorStatus *metav1.Status
	for {
		obj, _, err := streamDecoder.Decode(nil, &metav1.WatchEvent{})
		if err != nil {
			break
		}
		event, ok := obj.(*metav1.WatchEvent)
		if !ok {
			continue
		}
		if event.Type != string(watch.Error) {
			continue
		}
		decoded, _, err := scheme.Codecs.UniversalDeserializer().Decode(event.Object.Raw, nil, nil)
		if err != nil {
			t.Fatalf("client-side decode of the ERROR event payload failed: %v", err)
		}
		status, ok := decoded.(*metav1.Status)
		if !ok {
			t.Fatalf("decoded ERROR payload = %T, want *metav1.Status", decoded)
		}
		sawErrorStatus = status
		break
	}
	if sawErrorStatus == nil {
		t.Fatal("watch stream contained no decodable ERROR event")
	}
	if sawErrorStatus.Code != int32(http.StatusGone) {
		t.Fatalf("status code = %d, want %d", sawErrorStatus.Code, http.StatusGone)
	}
	if sawErrorStatus.Kind != "Status" || sawErrorStatus.APIVersion == "" {
		t.Fatalf("status TypeMeta = %s/%s, want kind Status with an apiVersion the client can decode",
			sawErrorStatus.APIVersion, sawErrorStatus.Kind)
	}
}

// Normal events decoded by a client must resolve to the typed resource with
// its kind/apiVersion intact — the versioning codec stamps them at encode time.
func TestServeWatchInjectsTypeMetaOnNormalEvents(t *testing.T) {
	cluster := &corev1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "c1"}}
	watcher := newSingleEventWatch(watch.Event{Type: watch.Added, Object: cluster})

	httpReq := httptest.NewRequestWithContext(context.Background(), "GET",
		"/apis/core.kubeclipper.io/v1/clusters?watch=true", http.NoBody)
	httpReq.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	ServeWatch(watcher, corev1.SchemeGroupVersion.WithKind("Cluster"),
		restful.NewRequest(httpReq), restful.NewResponse(recorder), 10*time.Second)

	event := decodeSingleWatchEvent(t, recorder.Body.Bytes(), string(watch.Added))
	decoded, _, err := scheme.Codecs.UniversalDeserializer().Decode(event.Object.Raw, nil, nil)
	if err != nil {
		t.Fatalf("client-side decode of the ADDED payload failed: %v", err)
	}
	got, ok := decoded.(*corev1.Cluster)
	if !ok {
		t.Fatalf("decoded payload = %T, want *corev1.Cluster", decoded)
	}
	if got.Name != "c1" {
		t.Fatalf("decoded cluster name = %q, want c1", got.Name)
	}
}

// Bookmark events must carry the resource GVK so clients can process them
// without a manual per-event stamp.
func TestServeWatchBookmarkCarriesGVK(t *testing.T) {
	cluster := &corev1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "c1", ResourceVersion: "42"}}
	watcher := newSingleEventWatch(watch.Event{Type: watch.Bookmark, Object: cluster})

	httpReq := httptest.NewRequestWithContext(context.Background(), "GET",
		"/apis/core.kubeclipper.io/v1/clusters?watch=true", http.NoBody)
	httpReq.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	ServeWatch(watcher, corev1.SchemeGroupVersion.WithKind("Cluster"),
		restful.NewRequest(httpReq), restful.NewResponse(recorder), 10*time.Second)

	event := decodeSingleWatchEvent(t, recorder.Body.Bytes(), string(watch.Bookmark))
	decoded, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(event.Object.Raw, nil, nil)
	if err != nil {
		t.Fatalf("client-side decode of the BOOKMARK payload failed: %v", err)
	}
	if gvk.Kind != "Cluster" || gvk.Group != corev1.SchemeGroupVersion.Group {
		t.Fatalf("bookmark payload gvk = %v, want core.kubeclipper.io/v1 Cluster", gvk)
	}
	if _, ok := decoded.(*corev1.Cluster); !ok {
		t.Fatalf("decoded payload = %T, want *corev1.Cluster", decoded)
	}
}

func decodeSingleWatchEvent(t *testing.T, body []byte, wantType string) *metav1.WatchEvent {
	t.Helper()
	frameReader := json.Framer.NewFrameReader(io.NopCloser(bytes.NewReader(body)))
	streamDecoder := streaming.NewDecoder(frameReader, scheme.Codecs.UniversalDeserializer())
	for {
		obj, _, err := streamDecoder.Decode(nil, &metav1.WatchEvent{})
		if err != nil {
			t.Fatalf("watch stream ended without a %q event: %v", wantType, err)
		}
		event, ok := obj.(*metav1.WatchEvent)
		if !ok {
			continue
		}
		if event.Type != wantType {
			continue
		}
		return event
	}
}
