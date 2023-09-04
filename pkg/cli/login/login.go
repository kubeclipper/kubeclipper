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

package login

import (
	"context"
	"fmt"
	"os"

	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"

	"github.com/spf13/cobra"

	"golang.org/x/term"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
  Login to the kubeclipper server and acquire access token.

  This command is the pre-operation of several cli commands, So if you encounter this error 'open /root/.kc/config: no such file or directory', you may need to execute the login command first.

  The command currently stores the results to the /root/.kc/config file by default.`
	loginExample = `
  # Login to the kubeclipper server
  kcctl login --host https://127.0.0.1 --username admin

  # Login to the kubeclipper server via passwd by cli
  kcctl login --host https://127.0.0.1 --username admin --password xxx

  Please read 'kcctl login -h' get more login flags.`
)

type LoginOptions struct {
	PrintFlags *printer.PrintFlags
	options.IOStreams
	Username string
	Password string
	Host     string
}

func NewLoginOptions(streams options.IOStreams) *LoginOptions {
	return &LoginOptions{
		PrintFlags: printer.NewPrintFlags(),
		IOStreams:  streams,
		Username:   "",
		Password:   "",
		Host:       "",
	}
}

func NewCmdLogin(streams options.IOStreams) *cobra.Command {

	o := NewLoginOptions(streams)
	cmd := &cobra.Command{
		Use:                   "login (--host | -H <host>) (--username | -u <username>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Login to the kubeclipper server",
		Long:                  longDescription,
		Example:               loginExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgs(cmd, args))
			utils.CheckErr(o.RunLogin())
		},
	}

	cmd.Flags().StringVarP(&o.Username, "username", "u", o.Username, "kubeclipper username")
	cmd.Flags().StringVarP(&o.Password, "password", "p", o.Password, "kubeclipper user password")
	cmd.Flags().StringVarP(&o.Host, "host", "H", o.Host, "kubeclipper server address, format as https://host")
	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("username")
	return cmd
}

func (l *LoginOptions) ValidateArgs(cmd *cobra.Command, args []string) error {
	if l.Host == "" {
		return utils.UsageErrorf(cmd, "--host must be valid")
	}
	if l.Username == "" {
		return utils.UsageErrorf(cmd, "--username must be valid")
	}
	if l.Password != "" {
		_, _ = fmt.Fprintf(l.IOStreams.Out, "WARNING! Using --password via the CLI is insecure.\n")
	}
	return nil
}

func (l *LoginOptions) RunLogin() error {
	if l.Password == "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			_, _ = fmt.Fprintf(l.IOStreams.Out, "Please input password for user %s: ", l.Username)
			pBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}
			l.Password = string(pBytes)
			_, _ = fmt.Fprintf(l.IOStreams.Out, "\n")
		}
	}
	// TODO use WithCAData insteadof WithInsecureSkipTLSVerify
	c, err := kc.NewClientWithOpts(kc.WithEndpoint(l.Host), kc.WithInsecureSkipTLSVerify())
	if err != nil {
		return err
	}

	resp, err := c.Login(context.TODO(), kc.LoginRequest{
		Username: l.Username,
		Password: l.Password,
	})
	if err != nil {
		return err
	}
	cfg, err := config.TryLoadFromFile(options.DefaultConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg = config.New()
	}
	newCfg := &config.Config{
		Servers: map[string]*config.Server{
			"default": {
				Server: l.Host,
				// TODO get ca form kc server
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*config.AuthInfo{
			l.Username: {
				Token: resp.AccessToken,
			},
		},
		CurrentContext: fmt.Sprintf("%s@default", l.Username),
		Contexts: map[string]*config.Context{
			fmt.Sprintf("%s@default", l.Username): {
				AuthInfo: l.Username,
				Server:   "default",
			},
		},
	}

	cfg.Merge(newCfg)

	return cfg.Dump()
}
