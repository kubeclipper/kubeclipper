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
	"regexp"
	"strings"
	"sync"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func PreCheck(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string) bool {
	return PreCheckError(name, sshConfig, streams, allNodes) == nil
}

// PreCheckError verifies sudo access and preserves the reason when the user declines to continue.
func PreCheckError(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string) error {
	logger.Infof("============>%s PRECHECK ...", name)
	if sshConfig.User == "root" {
		logger.Infof("============>%s PRECHECK OK!", name)
		return nil
	}
	var lastErr error
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
				lastErr = err
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
		return nil
	}

	_, _ = streams.Out.Write([]byte("Ignore this error, still exec cmd? Please input (yes/no)"))
	if utils.AskForConfirmation() {
		return nil
	}
	return fmt.Errorf("%s precheck failed: %w", name, lastErr)
}

// MultiNIC check node has multi NIC but node specify ip-detect flag.
func MultiNIC(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string, ipDetect string) bool {
	return MultiNICError(name, sshConfig, streams, allNodes, ipDetect) == nil
}

// MultiNICError verifies IP detection safety and preserves the reason when the user declines to continue.
func MultiNICError(name string, sshConfig *sshutils.SSH, streams options.IOStreams, allNodes []string, ipDetect string) error {
	logger.Infof("============>%s PRECHECK ...", name)
	if ipDetect != "" && ipDetect != autodetection.MethodFirst {
		logger.Infof("============>%s PRECHECK OK!", name)
		return nil
	}
	hasMultiNIC, errs := checkMultiNIC(sshConfig, allNodes)
	if firstErr := firstError(errs); firstErr != nil {
		return handleMultiNICError(name, streams, firstErr)
	}
	multiNICNodes := collectMultiNICNodes(allNodes, hasMultiNIC)
	if len(multiNICNodes) == 0 {
		logger.Infof("============>%s PRECHECK OK!", name)
		return nil
	}
	return handleMultiNICWarning(name, streams, multiNICNodes)
}

func checkMultiNIC(sshConfig *sshutils.SSH, allNodes []string) ([]bool, []error) {
	hasMultiNIC := make([]bool, len(allNodes))
	errs := make([]error, len(allNodes))
	wg := sync.WaitGroup{}
	for i, node := range allNodes {
		wg.Add(1)
		go func(idx int, host string) {
			defer wg.Done()
			result, err := sshutils.SSHCmd(sshConfig, host, `ip a|grep ": "|awk {'print $2'}|sed 's/://'`)
			if err != nil {
				if strings.Contains(err.Error(), "handshake failed: ssh: unable to authenticate, attempted methods [none password]") {
					errs[idx] = fmt.Errorf("passwd or user error while ssh '%s@%s',please try again", sshConfig.User, host)
				} else {
					errs[idx] = err
				}
				return
			}
			if result.ExitCode != 0 {
				errs[idx] = fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
				return
			}
			ifaces := strings.Split(result.Stdout, "\n")
			ifaces = sliceutil.RemoveString(ifaces, func(item string) bool {
				return item == ""
			})
			rifcae := filterLogicIface(ifaces)
			if len(rifcae) > 1 {
				hasMultiNIC[idx] = true
			}
		}(i, node)
	}
	wg.Wait()
	return hasMultiNIC, errs
}

func firstError(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func collectMultiNICNodes(allNodes []string, hasMultiNIC []bool) []string {
	var nodes []string
	for i, host := range allNodes {
		if i < len(hasMultiNIC) && hasMultiNIC[i] {
			nodes = append(nodes, host)
		}
	}
	return nodes
}

func handleMultiNICError(name string, streams options.IOStreams, firstErr error) error {
	logger.Error(firstErr)
	if options.AssumeYes {
		logger.Infof("skip this error,continue exec cmd")
		return nil
	}
	logger.Errorf("===========>%s PRECHECK FAILED!", name)
	fmt.Fprint(streams.Out, "Ignore this error, still exec cmd? Please input (yes/no)")
	if utils.AskForConfirmation() {
		return nil
	}
	return fmt.Errorf("%s precheck failed: %w", name, firstErr)
}

func handleMultiNICWarning(name string, streams options.IOStreams, multiNICNodes []string) error {
	logger.Warnf("node has multiple network interfaces, --ip-detect not specified (default: first-found may choose wrong interface):")
	for _, host := range multiNICNodes {
		logger.Warnf("  - [agent@%s]", host)
	}
	if options.AssumeYes {
		logger.Infof("skip this error,continue exec cmd")
		return nil
	}
	logger.Errorf("===========>%s PRECHECK FAILED!", name)
	fmt.Fprintln(streams.Out, "node has multi nic,and --ip-detect flag not specified,default ip "+
		"detect method is 'first-found',which maybe chose a wrong one,you can add --ip-detect flag to specify it.")
	fmt.Fprint(streams.Out, "Ignore this error, still exec cmd? Please input (yes/no)")
	if utils.AskForConfirmation() {
		return nil
	}
	return fmt.Errorf(
		"%s precheck failed: nodes have multiple network interfaces and "+
			"--ip-detect is first-found: %s", name, strings.Join(multiNICNodes, ", "),
	)
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
