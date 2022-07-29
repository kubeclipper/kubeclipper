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

package cmdutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func RunCmdWithContext(ctx context.Context, dryRun bool, command string, args ...string) (*ExecCmd, error) {
	ec := NewExecCmd(ctx, command, args...)
	logger.Debug("running command", zap.String("cmd", ec.String()))
	if dryRun {
		_, err := ec.stdOutBuf.WriteString("dry run command")
		return ec, err
	}
	// check context, get log file if conditions permit
	f, check, err := CheckContextAndGetStepLogFile(ctx)
	if err != nil {
		// detect context content and distinguish errors
		if check {
			logger.Error("get operation step log file failed: "+err.Error(),
				zap.String("operation", component.GetOperationID(ctx)),
				zap.String("step", component.GetStepID(ctx)),
				zap.String("cmd", ec.String()),
			)
		} else {
			// commands do not need to be logged
			logger.Debug("this command does not need to be logged", zap.String("cmd", ec.String()))
		}
	} else {
		// records the command that are being executed
		if _, err = f.Write([]byte(fmt.Sprintf("[%s] + %s\n\n", time.Now().Format(time.RFC3339), ec.String()))); err != nil {
			logger.Error("records execute the command error", zap.String("cmd", ec.String()))
		}
		// ignore the error
		defer f.Close()
		// Set the file descriptor to the receiver of the commands stdout and stderr, to synchronize output to log file.
		ec.SetStdoutMultiWriter(f)
		ec.SetStderrMultiWriter(f)
		logger.Debug("set log file to the receiver of the commands stdout and stderr, start sync log", zap.String("cmd", command))
	}
	doneCh := make(chan struct{})
	defer close(doneCh)
	//// set Setpgid=true to create new process group.
	//ec.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	//// kill all child process after context is done.
	//go func() {
	//	select {
	//	case <-ctx.Done():
	//		// Send signal iff the process is in execution state
	//		if ec.Cmd.Process == nil {
	//			logger.Debug("the current command is not running", zap.String("cmd", ec.String()))
	//			return
	//		}
	//		// If pid is less than -1, then sig is sent to every process in the process group whose ID is -pid.
	//		// https://man7.org/linux/man-pages/man2/kill.2.html
	//		err = syscall.Kill(-ec.Cmd.Process.Pid, syscall.SIGKILL)
	//		if err != nil {
	//			logger.Error("kill child process error", zap.String("cmd", ec.String()))
	//			return
	//		}
	//		logger.Debug("run command timeout,killed all child process", zap.String("cmd", ec.String()))
	//	case <-doneCh:
	//	}
	//}()
	if err = ec.Run(); err != nil {
		logger.Error("run command failed: "+err.Error(), zap.String("cmd", ec.String()))
		return ec, err
	}
	return ec, nil
}

func RunCmd(dryRun bool, command string, args ...string) (*ExecCmd, error) {
	return RunCmdWithContext(context.TODO(), dryRun, command, args...)
}

// CmdPipeline run pipe command like cat a | grep -i "bla"
func CmdPipeline(cmds ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
	// Require at least one command
	if len(cmds) < 1 {
		return nil, nil, nil
	}
	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}
	// Connect the output and error for the last command
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	// Start each command
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}
	// Wait for each command to complete
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}
	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
}

// CheckContextAndGetStepLogFile checks whether the context contains instance data.
// Caution: when check is true, you should close the file descriptor.
func CheckContextAndGetStepLogFile(ctx context.Context) (*os.File, bool, error) {
	opID := component.GetOperationID(ctx)
	stepID := component.GetStepID(ctx)
	opLog := component.GetOplog(ctx)
	if opID == "" || stepID == "" || opLog == nil {
		return nil, false, errors.New("prerequisites for recording operation logs are not complete, please check")
	}
	f, err := opLog.CreateStepLogFile(opID, stepID)
	if err != nil {
		return f, true, err
	}
	return f, true, nil
}

// CheckContextAndAppendStepLogFile checks whether the context contains instance data and append data to log file.
func CheckContextAndAppendStepLogFile(ctx context.Context, data []byte) (bool, error) {
	f, check, err := CheckContextAndGetStepLogFile(ctx)
	if err != nil {
		return check, err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return true, err
	}
	return true, nil
}
