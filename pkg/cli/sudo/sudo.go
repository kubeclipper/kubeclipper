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

package sudo

import (
	"fmt"
	"strings"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func PreCheck(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string) bool {
	logger.Infof("============>%s PRECHECK ...", name)
	if sshConfig.User == "root" {
		logger.Infof("============>%s PRECHECK OK!", name)
		return true
	}
	for {
		// need enter passwd to run sudo
		if sshConfig.Password == "" {
			_, _ = streams.Out.Write([]byte(fmt.Sprintf("ensure cmd exec success,need enter passwd for user '%s'. "+
				"Please input (user %s's password)", sshConfig.User, sshConfig.User)))
			passwd, err := utils.WaitInputPasswd()
			if err != nil {
				logger.V(2).Errorf("read passwd error: ", err.Error())
				continue
			}
			if passwd == "" {
				continue
			}
			_, _ = streams.Out.Write([]byte("\n"))
			sshConfig.Password = passwd
		}
		// check sudo access
		err := sshutils.CmdBatchWithSudo(sshConfig, allNodes, "id -u", func(result sshutils.Result, err error) error {
			if err != nil {
				if strings.Contains(err.Error(), "handshake failed: ssh: unable to authenticate, attempted methods [none password]") {
					return fmt.Errorf("passwd or user error while ssh '%s@%s',please try again", result.User, result.Host)
				}
				return err
			}
			if result.ExitCode != 0 {
				if strings.Contains(result.Stderr, "is not in the sudoers file") {
					return fmt.Errorf("user '%s@%s' is not in the sudoers file,please config it", result.User, result.Host)
				}

				if strings.Contains(result.Stderr, "incorrect password attempt") {
					return fmt.Errorf("passwd error for '%s@%s',please try again", result.User, result.Host)
				}
				return fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
			}
			return nil
		})

		if err != nil {
			logger.Error(err)
			logger.Errorf("===========>%s PRECHECK FAILED!", name)
			// if user can't access sudo,break
			if strings.Contains(err.Error(), "is not in the sudoers file") {
				_, _ = streams.Out.Write([]byte(err.Error() + "\n"))
				break
			}
			// if error is username or password incorrect,we reset it.
			if strings.Contains(err.Error(), "passwd or user error") {
				_, _ = streams.Out.Write([]byte(err.Error() + "\n"))
				sshConfig.Password = ""
			}
			continue
		}
		logger.Infof("============>%s PRECHECK OK!", name)
		return true
	}

	_, _ = streams.Out.Write([]byte("Ignore this error, still exec cmd? Please input (yes/no)"))
	return utils.AskForConfirmation()
}
