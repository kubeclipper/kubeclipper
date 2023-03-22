package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml"

	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

const (
	containerdDefaultConfig = "/etc/containerd/config.toml"
	dockerDefaultConfig     = "/etc/docker/daemon.json"
)

func AddOrRemoveInsecureRegistryToCRI(ctx context.Context, criType, registry string, add, dryRun bool) error {
	switch criType {
	case "containerd":
		return addOrRemoveContainerdInsecureRegistry(ctx, registry, add, dryRun)
	case "docker":
		return addOrRemoveDockerInsecureRegistry(ctx, registry, add, dryRun)
	default:
		return fmt.Errorf("%s CRI is not supported", criType)
	}
}

func addOrRemoveContainerdInsecureRegistry(ctx context.Context, registry string, add, dryRun bool) (err error) {
	if dryRun {
		return
	}
	info, err := os.Stat(containerdDefaultConfig)
	if err != nil {
		return
	}
	// load toml config file
	conf, err := toml.LoadFile(containerdDefaultConfig)
	if err != nil {
		return
	}
	var logMsg string
	insecureRegistries := conf.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "registry"}).(*toml.Tree)
	if add {
		// add registry table
		// if the key already exists, it will not be added again.
		insecureRegistries.SetPath([]string{"mirrors", registry, "endpoint"}, []string{fmt.Sprintf("http://%s", registry)})
		logMsg = fmt.Sprintf("write %s registry to %s", registry, containerdDefaultConfig)
	} else {
		// delete registry table
		if err = insecureRegistries.DeletePath([]string{"mirrors", registry}); err != nil {
			return
		}
		logMsg = fmt.Sprintf("delete %s registry from %s", registry, containerdDefaultConfig)
	}
	data, err := conf.ToTomlString()
	if err != nil {
		return
	}
	if err = os.WriteFile(containerdDefaultConfig, []byte(data), info.Mode()); err != nil {
		return
	}
	_, _ = cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(fmt.Sprintf("[%s] + %s \n", time.Now().Format(time.RFC3339), logMsg)))
	// Restart containerd by running systemctl, containerd does not restart existing containers.
	// Therefore, the normal running of existing containers is not affected.
	if _, err = cmdutil.RunCmdWithContext(ctx, dryRun, "bash", "-c", "systemctl daemon-reload && systemctl restart containerd"); err != nil {
		return
	}
	return
}

func addOrRemoveDockerInsecureRegistry(ctx context.Context, registry string, add, dryRun bool) (err error) {
	if dryRun {
		return
	}
	info, err := os.Stat(dockerDefaultConfig)
	if err != nil {
		return
	}
	fileData, err := os.ReadFile(dockerDefaultConfig)
	if err != nil {
		return
	}
	var data map[string]interface{}
	if err = json.Unmarshal(fileData, &data); err != nil {
		return
	}
	// initialize insecure-registries
	if data["insecure-registries"] == nil {
		data["insecure-registries"] = []interface{}{}
	}
	logMsg := fmt.Sprintf("delete %s registry from %s", registry, dockerDefaultConfig)
	insecureRegistries := data["insecure-registries"].([]interface{})
	for k, v := range insecureRegistries {
		if v.(string) == registry {
			// add insecure registry
			// if the key already exists, skip subsequent operations
			if add {
				return
			}
			// remove insecure registry, delete the slice specified key
			insecureRegistries = append(insecureRegistries[:k], insecureRegistries[k+1:]...)
		}
	}
	if add {
		insecureRegistries = append(insecureRegistries, registry)
		logMsg = fmt.Sprintf("write %s registry to %s", registry, dockerDefaultConfig)
	}
	data["insecure-registries"] = insecureRegistries
	newData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(dockerDefaultConfig, newData, info.Mode()); err != nil {
		return err
	}
	_, _ = cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(fmt.Sprintf("[%s] + %s \n", time.Now().Format(time.RFC3339), logMsg)))
	// Reload docker by running systemctl, docker does not restart and does not affect the created containers.
	if _, err = cmdutil.RunCmdWithContext(ctx, dryRun, "bash", "-c", "systemctl reload docker"); err != nil {
		return err
	}
	return nil
}
