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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func NewSSHClient(username, password, ip string, port int) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		Timeout:         5 * time.Second,
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
	}
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

type safeBuffer struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

func (w *safeBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}
func (w *safeBuffer) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Bytes()
}
func (w *safeBuffer) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer.Reset()
}

type LogicSSHWsSession struct {
	stdinPipe       io.WriteCloser
	comboOutput     *safeBuffer
	logBuff         *safeBuffer
	inputFilterBuff *safeBuffer
	session         *ssh.Session
	wsConn          *websocket.Conn
	isAdmin         bool
	IsFlagged       bool
}

func NewLoginSSHWSSession(cols, rows int, isAdmin bool, sshClient *ssh.Client, wsConn *websocket.Conn) (*LogicSSHWsSession, error) {
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}
	comboWriter := new(safeBuffer)
	logBuf := new(safeBuffer)
	inputBuf := new(safeBuffer)
	//ssh.stdout and stderr will write output into comboWriter
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
	return &LogicSSHWsSession{
		stdinPipe:       stdinP,
		comboOutput:     comboWriter,
		logBuff:         logBuf,
		inputFilterBuff: inputBuf,
		session:         sshSession,
		wsConn:          wsConn,
		isAdmin:         isAdmin,
		IsFlagged:       false,
	}, nil
}

// Close 关闭
func (sws *LogicSSHWsSession) Close() {
	if sws.session != nil {
		sws.session.Close()
	}
	if sws.logBuff != nil {
		sws.logBuff = nil
	}
	if sws.comboOutput != nil {
		sws.comboOutput = nil
	}
}
func (sws *LogicSSHWsSession) Start(quitChan chan struct{}) {
	go sws.receiveWsMsg(quitChan)
	go sws.sendComboOutput(quitChan)
}

// receiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (sws *LogicSSHWsSession) receiveWsMsg(exitCh chan struct{}) {
	wsConn := sws.wsConn
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				logger.Errorf("reading webSocket message failed due to error: %s", err.Error())
				return
			}
			//unmashal bytes into struct
			msgObj := wsMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				logger.Errorf("unmarshal websocket message failed due to error: %s, wsData is %s", err.Error(), string(wsData))
			}
			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := sws.session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						logger.Errorf("ssh pty change windows size failed due to error: %s", err.Error())
					}
				}
			case wsMsgCmd:
				//handle xterm.js stdin
				decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				if err != nil {
					logger.Errorf("websock cmd string base64 decoding failed due to error: %s", err.Error())
				}
				sws.sendWebsocketInputCommandToSSHSessionStdinPipe(decodeBytes)
			}
		}
	}
}

// sendWebsocketInputCommandToSshSessionStdinPipe
func (sws *LogicSSHWsSession) sendWebsocketInputCommandToSSHSessionStdinPipe(cmdBytes []byte) {
	if _, err := sws.stdinPipe.Write(cmdBytes); err != nil {
		logger.Errorf("ws cmd bytes write to ssh.stdin pipe failed due to error: %s", err.Error())
	}
}
func (sws *LogicSSHWsSession) sendComboOutput(exitCh chan struct{}) {
	wsConn := sws.wsConn
	//todo 优化成一个方法
	//tells other go routine quit
	defer setQuit(exitCh)
	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(100))
	//for range time.Tick(120 * time.Millisecond){}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if sws.comboOutput == nil {
				return
			}
			bs := sws.comboOutput.Bytes()
			if len(bs) > 0 {
				logger.Debugf("output data", zap.ByteString("data", bs))
				err := wsConn.WriteMessage(websocket.TextMessage, bs)
				if err != nil {
					logger.Errorf("ssh sending combo output to webSocket failed due to error: %s", err.Error())
				}
				_, err = sws.logBuff.Write(bs)
				if err != nil {
					logger.Errorf("combo output to logger buffer failed due to error: %s", err.Error())
				}
				sws.comboOutput.buffer.Reset()
			}
		case <-exitCh:
			return
		}
	}
}
func (sws *LogicSSHWsSession) Wait(quitChan chan struct{}, gracefulExitChan chan struct{}) {
	if err := sws.session.Wait(); err != nil {
		logger.Errorf("ssh session wait failed due to error: %s", err.Error())
		setQuit(quitChan)
	}
	logger.Debug("ssh session exit by remote peer")
	close(gracefulExitChan)
}
func (sws *LogicSSHWsSession) LogString() string {
	return sws.logBuff.buffer.String()
}
func setQuit(ch chan struct{}) {
	ch <- struct{}{}
}
