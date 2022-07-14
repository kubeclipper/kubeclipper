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

package restplus

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"github.com/kubeclipper/kubeclipper/pkg/utils/wssstream"

	"github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"

	"k8s.io/apimachinery/pkg/runtime"
)

func ServeWatch(watcher watch.Interface, gvk schema.GroupVersionKind, request *restful.Request, response *restful.Response, timeout time.Duration) {
	defer watcher.Stop()

	serializer, err := negotiation.NegotiateOutputMediaTypeStream(request.Request, scheme.Codecs, negotiation.DefaultEndpointRestrictions)
	if err != nil {
		HandleInternalError(response, request, err)
		return
	}
	if serializer.StreamSerializer.Framer == nil {
		HandleInternalError(response, request, fmt.Errorf("no framer defined for %q available for embedded encoding", serializer.MediaType))
		return
	}
	framer := serializer.StreamSerializer.Framer.NewFrameWriter(response.ResponseWriter)

	e := streaming.NewEncoder(framer, scheme.Encoder)

	mediaType := serializer.MediaType
	if mediaType != "application/json" {
		mediaType += ";stream=watch"
	}

	if wssstream.IsWebSocketRequest(request.Request) {
		response.Header().Set("Content-Type", mediaType)
		handlerWebsocket(watcher, request, response, serializer.EncodesAsText)
		return
	}

	flusher, ok := response.ResponseWriter.(http.Flusher)
	if !ok {
		err := fmt.Errorf("unable to start watch - can't get http.Flusher: %#v", response.ResponseWriter)
		HandleInternalError(response, request, err)
		return
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	// begin the stream
	response.Header().Set("Content-Type", mediaType)
	response.Header().Set("Transfer-Encoding", "chunked")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()

	var unknown runtime.Unknown
	internalEvent := &metav1.InternalEvent{}
	outEvent := &metav1.WatchEvent{}
	buf := &bytes.Buffer{}
	ch := watcher.ResultChan()
	done := request.Request.Context().Done()

	for {
		select {
		case <-done:
			logger.Debug("done channel receive")
			return
		case <-timer.C:
			logger.Debug("timer receive")
			return
		case event, ok := <-ch:
			if !ok {
				// End of results.
				logger.Debug("watch channel stop")
				return
			}
			obj := event.Object
			if event.Type == watch.Bookmark {
				obj.GetObjectKind().SetGroupVersionKind(gvk)
			}
			if err := scheme.Encoder.Encode(obj, buf); err != nil {
				HandleInternalError(response, request, fmt.Errorf("unable to encode watch object %T: %v", obj, err))
				return
			}
			unknown.Raw = buf.Bytes()
			event.Object = &unknown
			*outEvent = metav1.WatchEvent{}
			*internalEvent = metav1.InternalEvent(event)
			err := metav1.Convert_v1_InternalEvent_To_v1_WatchEvent(internalEvent, outEvent, nil)
			if err != nil {
				HandleInternalError(response, request, fmt.Errorf("unable to convert watch object: %v", err))
				return
			}
			if err := e.Encode(outEvent); err != nil {
				HandleInternalError(response, request, fmt.Errorf("unable to encode watch object %T: %v (%#v)", outEvent, err, e))
				return
			}
			if len(ch) == 0 {
				flusher.Flush()
			}

			buf.Reset()
		}
	}
}

func handlerWebsocket(watcher watch.Interface, request *restful.Request, response *restful.Response, encodesAsText bool) {
	upgrade := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024 * 1024 * 10,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			HandlerErrorWithCustomCode(response, request, status, status, "websocket error", reason)
		},
	}
	wsConn, err := upgrade.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		logger.Error("upgrade http request failed", zap.Error(err))
		return
	}
	defer wsConn.Close()

	done := make(chan struct{})
	go func() {
		defer HandlerCrash()
		for {
			resetTimeout(wsConn, 0)
			if _, _, err := wsConn.ReadMessage(); err != nil {
				logger.Error("websocket read message error", zap.Error(err))
				break
			}
		}
		close(done)
	}()

	// Time allowed to write a message to the peer.
	writeWait := 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait := 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod := (pongWait * 9) / 10
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	wsConn.SetPongHandler(func(appData string) error {
		logger.Debug("receive websocket pong message")
		_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	wsConn.SetCloseHandler(func(code int, text string) error {
		logger.Debug("receive close websocket message", zap.Int("code", code), zap.String("text", text))
		return nil
	})

	var unknown runtime.Unknown
	internalEvent := &metav1.InternalEvent{}
	buf := &bytes.Buffer{}
	streamBuf := &bytes.Buffer{}
	ch := watcher.ResultChan()

	for {
		select {
		case <-done:
			logger.Debug("receive done channel")
			return
		case <-ticker.C:
			_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("write websocket ping message error", zap.Error(err))
				return
			}
			logger.Debug("send websocket ping message")
		case event, ok := <-ch:
			if !ok {
				// End of results.
				return
			}
			obj := event.Object
			if err := scheme.Encoder.Encode(obj, buf); err != nil {
				logger.Error("unable to encode watch object", zap.Any("object", obj), zap.Error(err))
				return
			}
			// ContentType is not required here because we are defaulting to the serializer
			// type
			unknown.Raw = buf.Bytes()
			event.Object = &unknown
			outEvent := &metav1.WatchEvent{}
			*internalEvent = metav1.InternalEvent(event)
			err := metav1.Convert_v1_InternalEvent_To_v1_WatchEvent(internalEvent, outEvent, nil)
			if err != nil {
				logger.Error("unable to convert watch object", zap.Error(err))
				return
			}
			if err := scheme.Encoder.Encode(outEvent, streamBuf); err != nil {
				logger.Error("unable to encode event", zap.Error(err))
				return
			}
			_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsConn.WriteMessage(websocket.TextMessage, streamBuf.Bytes()); err != nil {
				logger.Error("unable to write message", zap.Error(err))
				return
			}
			buf.Reset()
			streamBuf.Reset()
		}
	}
}

func resetTimeout(ws *websocket.Conn, timeout time.Duration) {
	if timeout > 0 {
		_ = ws.SetReadDeadline(time.Now().Add(timeout))
	}
}
