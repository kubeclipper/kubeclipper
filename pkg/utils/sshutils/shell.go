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
	"bytes"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"

	MagicExecShell = "(clear && bash) || (clear && ash) || (clear && sh) || bash || ash || sh"
)

type wsMsg struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// connect to ssh server using ssh session.
type SSHConn struct {
	// calling Write() to write data into ssh server
	StdinPipe io.WriteCloser
	// Write() be called to receive data from ssh server
	ComboOutput *wsBufferWriter
	Session     *ssh.Session
}

// setup ssh shell session
// set Session and StdinPipe here,
// and the Session.Stdout and Session.Sdterr are also set.
func NewSSHConn(cols, rows int, sshClient *ssh.Client) (*SSHConn, error) {
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	// we set stdin, then we can write data to ssh server via this stdin.
	// but, as for reading data from ssh server, we can set Session.Stdout and Session.Stderr
	// to receive data from ssh server, and write back to somewhere.
	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}
	comboWriter := new(wsBufferWriter)
	// ssh.stdout and stderr will write output into comboWriter
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	return &SSHConn{StdinPipe: stdinP, ComboOutput: comboWriter, Session: sshSession}, nil
}
func (s *SSHConn) Close() {
	if s.Session != nil {
		s.Session.Close()
	}
}

// copy data from WebSocket to ssh server
// and copy data from ssh server to WebSocket
// write data to WebSocket
// the data comes from ssh server.
type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// flushComboOutput flush ssh.session combine output into websocket response
// func flushComboOutput(w *wsBufferWriter, wsConn *websocket.Conn) error {
//	if w.buffer.Len() != 0 {
//		err := wsConn.WriteMessage(websocket.TextMessage, w.buffer.Bytes())
//		if err != nil {
//			return err
//		}
//		w.buffer.Reset()
//	}
//	return nil
// }
