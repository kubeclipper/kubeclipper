package externalcluster

import (
	"fmt"
	"strconv"
	"time"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

type GeneralOption struct {
	ProviderName string `json:"providerName"`
	// pod-agent/node-agent
	FedType string `json:"fedType"`
	// node region default value: "default"
	NodeRegion string `json:"nodeRegion,omitempty"`
	SSHConfig  *SSH   `json:"sshConfig"`
}

type SSH struct {
	User              string         `json:"user" yaml:"user,omitempty"`
	Port              int            `json:"port" yaml:"port,omitempty"`
	Password          string         `json:"password,omitempty" yaml:"password,omitempty"`
	PkDataEncode      string         `json:"pkDataEncode,omitempty" yaml:"pkDataEncode,omitempty"`
	ConnectionTimeout *time.Duration `json:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`
}

func (s *GeneralOption) ReadEntity(cm *v1.ConfigMap) error {
	if v, ok := cm.Data["providerName"]; ok {
		s.ProviderName = v
	}
	if v, ok := cm.Data["fedType"]; ok {
		s.FedType = v
	}
	if v, ok := cm.Data["nodeRegion"]; ok {
		s.NodeRegion = v
	}

	if s.ProviderName == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if s.FedType == "" {
		return fmt.Errorf("fed type cannot be empty")
	}
	if s.NodeRegion == "" {
		return fmt.Errorf("node region cannot be empty")
	}
	s.SSHConfig = &SSH{}
	return s.SSHConfig.ReadEntity(cm)
}

func (s *SSH) Convert() *sshutils.SSH {
	return &sshutils.SSH{
		User:           s.User,
		Password:       s.Password,
		PrivateKeyData: s.PkDataEncode,
		Port:           s.Port,
	}
}

func (s *SSH) ReadEntity(cm *v1.ConfigMap) error {
	if v, ok := cm.Data["user"]; ok {
		s.User = v
	}
	if v, ok := cm.Data["password"]; ok {
		s.Password = v
	}
	if v, ok := cm.Data["pkDataEncode"]; ok {
		s.PkDataEncode = v
	}
	if v, ok := cm.Data["port"]; ok {
		p, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			return err
		}
		s.Port = int(p)
	}
	if s.User == "" {
		return fmt.Errorf("ssh user cannot be empty")
	}
	if s.Password == "" && s.PkDataEncode == "" {
		return fmt.Errorf("ssh 'password' and 'pk data' at least one is required")
	}
	if s.Port == 0 {
		return fmt.Errorf("ssh port cannot be empty")
	}

	return nil
}
