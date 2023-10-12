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
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/olekukonko/tablewriter"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"golang.org/x/crypto/ssh"
)

type Result struct {
	User     string
	Host     string
	Cmd      string
	PrintCmd string
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r Result) Short() string {
	return fmt.Sprintf("(run `%s` on %s@%s)", r.PrintCmd, r.User, r.Host)
}

// Deprecated
// need not use this method to check error,will automic check when command run.
func (r Result) Error() error {
	return r.error()
}

func (r Result) error() error {
	if r.ExitCode == 0 {
		return nil
	}
	if r.Stderr != "" {
		return fmt.Errorf("%s exitcode %v msg: %s", r.Short(), r.ExitCode, r.Stderr)
	}
	return fmt.Errorf("%s exitcode %v msg: %s", r.Short(), r.ExitCode, r.Stdout)
}

// Deprecated
// use Table to replace
func (r Result) String() string {
	return r.Table()
}

// Table format as table
func (r Result) Table() string {
	buff := new(bytes.Buffer)

	table := tablewriter.NewWriter(buff)
	table.SetHeader([]string{"USER", "HOST", "COMMAND", "STDOUT", "STDERR", "EXIT CODE"})
	data := [][]string{
		{r.User, r.Host, r.PrintCmd, r.Stdout, r.Stderr, strconv.Itoa(r.ExitCode)},
	}

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
	return "\n" + buff.String() + "\n"
}

func (r Result) StdoutToString(place string) string {
	return strings.ReplaceAll(r.Stdout, "\n", place)
}

type SSHRunCmd func(sshConfig *SSH, host, cmd string) (Result, error)

// SSHCmdWithSudo  try to run cmd with sudo.
func SSHCmdWithSudo(sshConfig *SSH, host, cmd string) (Result, error) {
	var (
		sudoCmd = cmd
		err     error
	)
	// add sudo prefix if we need
	if !SSHToCmd(sshConfig, host) {
		result := Result{
			User: sshConfig.User,
			Host: host,
			Cmd:  cmd,
		}
		sudoCmd, err = fillCmd(sshConfig, cmd)
		if err != nil {
			return result, err
		}
	}
	return SSHCmd(sshConfig, host, sudoCmd)
}

// SSHCmd synchronously SSHs to a node running on provider and runs cmd. If there
// is no error performing the SSH, the stdout, stderr, and exit code are
// returned.
func SSHCmd(sshConfig *SSH, host, cmd string) (Result, error) {
	// if caller don't provide enough config for run ssh cmd，change to run cmd by os.exec on localhost.
	// only for aio deploy now.
	if SSHToCmd(sshConfig, host) {
		return RunCmdAsSSH(cmd)
	}
	stdout, stderr, code, err := runSSHCommand(sshConfig, host, cmd)
	result := Result{
		User:     sshConfig.User,
		Host:     host,
		Cmd:      cmd,
		PrintCmd: printCmd(sshConfig.Password, cmd),
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: code,
	}
	logger.V(5).Infof(result.Table())

	if err != nil {
		return result, err
	}
	// check exit code again when run cmd success
	return result, result.error()
}

// runSSHCommand returns the stdout, stderr, and exit code from running cmd on
// host as specific user, along with any SSH-level error.
func runSSHCommand(sshConfig *SSH, host, cmd string) (stdout, stderr string, exitcode int, err error) {
	pCmd := printCmd(sshConfig.Password, cmd)
	client, err := sshConfig.NewClient(host)
	if err != nil {
		return "", "", 0, err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", "", 0, err
	}
	defer session.Close()

	// Run the command.
	var bout, berr bytes.Buffer
	session.Stdout, session.Stderr = &bout, &berr
	if err = session.Run(cmd); err != nil {
		// Check whether the command failed to run or didn't complete.
		if exiterr, ok := err.(*ssh.ExitError); ok {
			// If we got an ExitError and the exit code is nonzero, we'll
			// consider the SSH itself successful (just that the command run
			// errored on the host).
			if exitcode = exiterr.ExitStatus(); exitcode != 0 {
				err = nil
			}
		} else {
			// Some other kind of error happened (e.g. an IOError); consider the
			// SSH unsuccessful.
			err = fmt.Errorf("failed running `%s` on %s@%s: '%v'", pCmd, sshConfig.User, host, err)
		}
	}
	return bout.String(), berr.String(), exitcode, err
}

type Walk func(result Result, err error) error

func DefaultWalk(result Result, err error) error {
	if err != nil {
		return err
	}
	return result.Error()
}

// CmdBatch parallel run cmd on many hosts
func CmdBatch(sshConfig *SSH, hosts []string, cmd string, walk Walk) error {
	return doCmdBatch(sshConfig, hosts, cmd, walk, SSHCmd)
}

// CmdBatchWithSudo parallel run cmd with sudo on many hosts
func CmdBatchWithSudo(sshConfig *SSH, hosts []string, cmd string, walk Walk) error {
	return doCmdBatch(sshConfig, hosts, cmd, walk, SSHCmdWithSudo)
}

func doCmdBatch(sshConfig *SSH, hosts []string, cmd string, walk Walk, fn SSHRunCmd) error {
	var (
		errCh  = make(chan error, len(hosts))
		stopCh = make(chan struct{})
		wg     sync.WaitGroup
	)
	wg.Add(len(hosts))
	for _, host := range hosts {
		go func(host string) {
			defer wg.Done()
			err := walk(fn(sshConfig, host, cmd))
			if err != nil {
				errCh <- err
			}
		}(host)
	}
	// new goroutine to wait all host finish
	go func() {
		defer func() {
			close(errCh)
			close(stopCh)
		}()
		wg.Wait()
		stopCh <- struct{}{}
	}()
	// return when omit at lease one error or all host finished
	select {
	case <-stopCh:
		return nil
	case err := <-errCh:
		return err
	}
}
