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

package natsio

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type NatsOptions struct {
	External bool          `yaml:"external" json:"external" mapstructure:"external"`
	Client   ClientOptions `yaml:"client" json:"client" mapstructure:"client"`
	Server   ServerOptions `yaml:"server" json:"server" mapstructure:"server"`
	Auth     AuthOptions   `yaml:"auth" json:"auth" mapstructure:"auth"`
}

type ClientOptions struct {
	ServerAddress     []string `yaml:"serverAddress" json:"serverAddress"`
	SubjectSuffix     string   `yaml:"subjectSuffix" json:"subjectSuffix"`
	QueueGroupName    string   `yaml:"queueGroupName" json:"queueGroupName"`
	NodeReportSubject string   `yaml:"nodeReportSubject" json:"nodeReportSubject"`
	TimeOutSeconds    int      `yaml:"timeOutSeconds" json:"timeOutSeconds"`
	// ReconnectWait sets the time to backoff after attempting a reconnect
	// to a server that we were already connected to previously.
	ReconnectInterval time.Duration `yaml:"reconnectInterval" json:"reconnectInterval"`
	// MaxReconnect sets the number of reconnect attempts that will be
	// tried before giving up. If negative, then it will never give up
	// trying to reconnect.
	MaxReconnect int `yaml:"maxReconnect" json:"maxReconnect"`
	// PingInterval is the period at which the client will be sending ping
	// commands to the server, disabled if 0 or negative.
	PingInterval time.Duration `yaml:"pingInterval" json:"pingInterval"`
	// MaxPingsOut is the maximum number of pending ping commands that can
	// be awaiting a response before raising an ErrStaleConnection error.
	MaxPingsOut int    `yaml:"maxPingsOut" json:"maxPingsOut"`
	TLSCaPath   string `yaml:"tlsCaPath" json:"tlsCaPath"`
	TLSCertPath string `yaml:"tlsCertPath" json:"tlsCertPath"`
	TLSKeyPath  string `yaml:"tlsKeyPath" json:"tlsKeyPath"`
}

type ServerOptions struct {
	Host        string         `yaml:"host" json:"host"`
	Port        int            `yaml:"port" json:"port"`
	Cluster     ClusterOptions `yaml:"cluster" json:"cluster" mapstructure:"cluster"`
	TLSCaPath   string         `yaml:"tlsCaPath" json:"tlsCaPath"`
	TLSCertPath string         `yaml:"tlsCertPath" json:"tlsCertPath"`
	TLSKeyPath  string         `yaml:"tlsKeyPath" json:"tlsKeyPath"`
}

type ClusterOptions struct {
	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port"`
	LeaderHost string `yaml:"leaderHost" json:"leader_host"`
}

type AuthOptions struct {
	UserName string `yaml:"username" json:"user_name"`
	Password string `yaml:"password" json:"password"`
}

func NewOptions() *NatsOptions {
	return &NatsOptions{
		Client: ClientOptions{
			ServerAddress:     []string{"127.0.0.1:9889"},
			SubjectSuffix:     "k8s-installer",
			QueueGroupName:    "kubeclipper-agent-report-queue",
			NodeReportSubject: "kubeclipper-agent-report-subject",
			TimeOutSeconds:    10,
			ReconnectInterval: 3 * time.Second,
			MaxReconnect:      600,
			PingInterval:      2 * time.Minute,
			MaxPingsOut:       2,
			TLSCaPath:         "",
			TLSCertPath:       "",
			TLSKeyPath:        "",
		},
		Server: ServerOptions{
			Host: "0.0.0.0",
			Port: 9889,
			Cluster: ClusterOptions{
				Host:       "0.0.0.0",
				Port:       9890,
				LeaderHost: "127.0.0.1",
			},
			TLSCaPath:   "",
			TLSCertPath: "",
			TLSKeyPath:  "",
		},
		Auth: AuthOptions{
			UserName: "username",
			Password: "password",
		},
	}
}

func (s *NatsOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&s.Client.ServerAddress, "mq-server-address", s.Client.ServerAddress,
		"message queue server address e.g abc.com or IP:PORT Default 127.0.0.1:9889")
	fs.StringVar(&s.Server.Host, "mq-server-host", s.Server.Host, "message queue server bind address. "+
		"Default is 0.0.0.0 listen for all interface")
	fs.IntVar(&s.Server.Port, "mq-server-port", s.Server.Port, "message queue server listening on port. Default 9889")
	fs.StringVar(&s.Client.SubjectSuffix, "mq-subject-suffix", s.Client.SubjectSuffix, "message queue subject suffix")
	fs.StringVar(&s.Client.QueueGroupName, "mq-queue-group-name", s.Client.QueueGroupName,
		"queue group name used to distribute message to one of a group listener member")
	fs.StringVar(&s.Client.NodeReportSubject, "mq-node-report-subject", s.Client.NodeReportSubject,
		"subject channel for node status report message queue channel")
	fs.IntVar(&s.Client.TimeOutSeconds, "mq-timeout-sec", s.Client.TimeOutSeconds, "mq connect timeout seconds")
	fs.DurationVar(&s.Client.ReconnectInterval, "mq-reconnect-interval", s.Client.ReconnectInterval,
		"message queue reconnect interval")
	fs.IntVar(&s.Client.MaxReconnect, "mq-max-connect", s.Client.MaxReconnect, ""+
		"MaxReconnect sets the number of reconnect attempts that will be tried before giving up. "+
		"If negative, then it will never give up trying to reconnect.")
	fs.StringVar(&s.Auth.UserName, "mq-username", s.Auth.UserName,
		"message queue auth username when auth mode is basic")
	fs.StringVar(&s.Auth.Password, "mq-password", s.Auth.Password,
		"message queue password when auth mode is basic")
	fs.StringVar(&s.Client.TLSCaPath, "mq-client-ca-cert", s.Client.TLSCaPath,
		"message queue ca cert file path")
	fs.StringVar(&s.Client.TLSCertPath, "mq-client-cert", s.Client.TLSCertPath,
		"message queue client cert file path")
	fs.StringVar(&s.Client.TLSKeyPath, "mq-client-cert-key", s.Client.TLSKeyPath,
		"message queue client cert key file path")
	fs.StringVar(&s.Server.Cluster.Host, "mq-cluster-host", s.Server.Cluster.Host, ""+
		"mq cluster host addr, only used in mq server")
	fs.IntVar(&s.Server.Cluster.Port, "mq-cluster-port", s.Server.Cluster.Port, ""+
		"mq cluster host addr port, only used in mq server")
	fs.StringVar(&s.Server.Cluster.LeaderHost, "mq-cluster-leader", s.Server.Cluster.LeaderHost, ""+
		"mq cluster leader addr, format at ip:port. only used in mq server")
	fs.StringVar(&s.Server.TLSCaPath, "mq-server-ca-cert", s.Server.TLSCaPath,
		"message queue ca cert file path")
	fs.StringVar(&s.Server.TLSCertPath, "mq-server-cert", s.Server.TLSCertPath,
		"message queue server cert file path")
	fs.StringVar(&s.Server.TLSKeyPath, "mq-server-cert-key", s.Server.TLSKeyPath,
		"message queue server cert key file path")
}

func (s *NatsOptions) Validate() []error {
	if s == nil {
		return nil
	}
	var err []error
	if !s.External && s.Server.Cluster.LeaderHost != "" {
		if parts := strings.Split(s.Server.Cluster.LeaderHost, ":"); len(parts) != 2 {
			err = append(err, fmt.Errorf("leader host %s must be ip:port", s.Server.Cluster.LeaderHost))
		}
	}
	if len(s.Client.ServerAddress) == 0 {
		err = append(err, fmt.Errorf("at least have one server address"))
	}
	for _, str := range s.Client.ServerAddress {
		if parts := strings.Split(str, ":"); len(parts) != 2 {
			err = append(err, fmt.Errorf("%s must be ip:port", str))
		}
	}
	return err
}

func (s *NatsOptions) CombineUsernameAndPassword() string {
	return fmt.Sprintf("%s:%s", s.Auth.UserName, s.Auth.Password)
}

func (s *NatsOptions) CombineServerAddress() string {
	return strings.Join(s.Client.ServerAddress, ",")
}

func (s *NatsOptions) GetConnectionString() string {
	return s.CombineServerAddress()
}
