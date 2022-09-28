package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, ciName, version, CheckCSIInstall), &CheckInstall{}); err != nil {
		panic(err)
	}
}

var (
	_ component.StepRunnable = (*CheckInstall)(nil)
)

const (
	ciName           = "check-install"
	ManifestsDir     = "/tmp"
	CSITestFile      = "csi-test.yaml"
	CheckCSIInstall  = "CheckCSIInstall"
	CheckCSITemplate = `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: csi-test-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: {{.StorageClassName}}
`
)

type CheckInstall struct {
	CsiType string
}

func (c *CheckInstall) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	// check install csi success or not
	if err := utils.RetryFunc(ctx, opts, 10*time.Second, "checkCSIInstall", c.checkCSIInstall); err != nil {
		return nil, err
	}
	// delete pod, pvc, pv
	_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "delete", "-f", ManifestsDir+"/."+c.CsiType+"/"+"csi-test.yaml")
	if err != nil {
		logger.Warnf("delete %s-test resources failed: %s", c.CsiType, err.Error())
	}

	return nil, err
}

func (c *CheckInstall) checkCSIInstall(ctx context.Context, opts component.Options) error {
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "get", "pvc", "csi-test-pvc")
	if err != nil {
		return err
	}
	outs := strings.Split(ec.StdOut(), "\n")
	if len(outs) > 1 {
		if strings.Contains(outs[1], "Bound") {
			return nil
		}
	}
	return fmt.Errorf("%s installation not completed", c.CsiType)
}

func (c *CheckInstall) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (c *CheckInstall) NewInstance() component.ObjectMeta {
	return &CheckInstall{}
}

// GetCheckCSIInstall get the common step
func GetCheckCSIInstall(nodes component.NodeList, csiType string) (v1.Step, error) {
	checkInstall := &CheckInstall{
		CsiType: csiType,
	}
	aData, err := json.Marshal(checkInstall)
	if err != nil {
		return v1.Step{}, err
	}
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "checkInstallCSI",
		Timeout:    metav1.Duration{Duration: 3 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      utils.UnwrapNodeList(nodes),
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, ciName, version, CheckCSIInstall),
				CustomCommand: aData,
			},
		},
	}, nil
}
