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
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
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

var (
	errorMultiNIC = errors.New("node has multi nic bug not specify --ip-detect flag")
)

// MultiNIC check node has multi NIC but node specify ip-detect flag.
func MultiNIC(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string, ipDetect string) bool {
	logger.Infof("============>%s PRECHECK ...", name)
	if ipDetect != "" && ipDetect != autodetection.MethodFirst {
		logger.Infof("============>%s PRECHECK OK!", name)
		return true
	}
	// check iface list
	// ifconfig|grep ": "|awk {'print $1'}|sed 's/://'
	err := sshutils.CmdBatch(sshConfig, allNodes, `ip a|grep ": "|awk {'print $2'}|sed 's/://'`, func(result sshutils.Result, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), "handshake failed: ssh: unable to authenticate, attempted methods [none password]") {
				return fmt.Errorf("passwd or user error while ssh '%s@%s',please try again", result.User, result.Host)
			}
			return err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
		}
		ifaces := strings.Split(result.Stdout, "\n")
		ifaces = sliceutil.RemoveString(ifaces, func(item string) bool {
			return item == ""
		})
		rifcae := filterLogicIface(ifaces)

		if len(rifcae) > 1 {
			return errorMultiNIC
		}
		return nil
	})

	if err != nil {
		logger.Error(err)
		if options.AssumeYes {
			logger.Infof("skip this error,continue exec cmd")
			return true
		}
		logger.Errorf("===========>%s PRECHECK FAILED!", name)
		if errors.Is(err, errorMultiNIC) {
			_, _ = streams.Out.Write([]byte("node has multi nic,and --ip-detect flag not specified,default ip " +
				"detect method is 'first-found',which maybe chose a wrong one,you can add --ip-detect flag to specify it." + "\n"))
		}
		_, _ = streams.Out.Write([]byte("Ignore this error, still exec cmd? Please input (yes/no)"))
		return utils.AskForConfirmation()
	}

	logger.Infof("============>%s PRECHECK OK!", name)
	return true
}

func filterLogicIface(ifcaes []string) []string {
	if len(autodetection.DefaultInterfacesToExclude) == 0 {
		return ifcaes
	}
	excludeRegexp, _ := regexp.Compile("(" + strings.Join(autodetection.DefaultInterfacesToExclude, ")|(") + ")")
	ret := make([]string, 0, len(ifcaes))
	for _, ifcae := range ifcaes {
		if !excludeRegexp.MatchString(ifcae) {
			ret = append(ret, ifcae)
		}
	}
	return ret
}
