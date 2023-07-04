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

package sshutils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// ctrl+d to close terminal
	endOfTransmission = "\u0004"
)

type Terminaler struct {
	config *rest.Config
}

func NewTerminaler(config *rest.Config) *Terminaler {
	return &Terminaler{config: config}
}

func (t *Terminaler) StartProcess(namespace, podName, containerName string, cmd []string, ptyHandler PtyHandler) error {
	rc, err := t.coreV1RestClient()
	if err != nil {
		return err
	}
	req := rc.Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(
			&corev1.PodExecOptions{
				Container: containerName,
				Command:   cmd,
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			},
			scheme.ParameterCodec,
		)

	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		return errors.WithMessage(err, "NewSPDYExecutor")
	}
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:             ptyHandler,
		Stdout:            ptyHandler,
		Stderr:            ptyHandler,
		TerminalSizeQueue: ptyHandler,
		Tty:               true,
	})
	return errors.WithMessage(err, "Stream")
}

func (t *Terminaler) coreV1RestClient() (*rest.RESTClient, error) {
	cfg := *t.config
	cfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	cfg.APIPath = "/api"
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	return rest.RESTClientFor(&cfg)
}

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// TerminalSession implements PtyHandler (using a SockJS connection)
type TerminalSession struct {
	Conn     *websocket.Conn
	SizeChan chan remotecommand.TerminalSize
}

// Next handles pty->process resize events
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Next() *remotecommand.TerminalSize {
	size := <-t.SizeChan
	if size.Height == 0 && size.Width == 0 {
		return nil
	}
	return &size
}

// Read handles pty->process messages (stdin, resize)
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Read(p []byte) (int, error) {
	var msg wsMsg
	err := t.Conn.ReadJSON(&msg)
	if err != nil {
		return copy(p, endOfTransmission), err
	}
	switch msg.Type {
	case wsMsgCmd:
		decodeString, err := base64.StdEncoding.DecodeString(msg.Cmd)
		if err != nil {
			return copy(p, endOfTransmission), err
		}
		return copy(p, decodeString), nil
	case wsMsgResize:
		t.SizeChan <- remotecommand.TerminalSize{Width: uint16(msg.Cols), Height: uint16(msg.Rows)}
		return 0, nil
	default:
		return copy(p, endOfTransmission), fmt.Errorf("unknown message type '%s'", msg.Type)
	}
}

// Write handles process->pty stdout
// Called from remotecommand whenever there is any output
func (t TerminalSession) Write(p []byte) (int, error) {
	_ = t.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := t.Conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t TerminalSession) Toast(p string) error {
	msg, err := json.Marshal(wsMsg{
		Type: "toast",
		Cmd:  p,
	})
	if err != nil {
		return err
	}
	_ = t.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return t.Conn.WriteMessage(websocket.TextMessage, msg)
}

func (t TerminalSession) Close(status uint32, reason string) {
	logger.Infof("[close] status:%v reason:%s", status, reason)
	close(t.SizeChan)
	_ = t.Conn.Close()
}
