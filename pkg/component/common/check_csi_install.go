package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

func init() {
	c := &CSIHealthCheck{}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, CsiHealthCheck, version, HealthCheck), c); err != nil {
		panic(err)
	}
	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat, CsiHealthCheck, version, HealthCheck), c); err != nil {
		panic(err)
	}
}

var (
	_ component.StepRunnable   = (*CSIHealthCheck)(nil)
	_ component.TemplateRender = (*CSIHealthCheck)(nil)
)

const (
	version                = "v1"
	CsiHealthCheck         = "csi-healthcheck"
	ManifestsDir           = "/tmp/.csi-healthcheck"
	checkCSIHealthFile     = "csi-test.yaml"
	HealthCheck            = "CheckCSIHealth"
	CSIHealthCheckTemplate = `
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

type CSIHealthCheck struct {
	StorageClassName string
}

func (c *CSIHealthCheck) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	filePath := filepath.Join(ManifestsDir, checkCSIHealthFile)
	// delete the resource that will be created later
	_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "delete", "-f", filePath)
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		logger.Warnf("kubectl delete %s failed: %s", filePath, err.Error())
	}
	// create pvc
	_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "apply", "-f", filePath)
	if err != nil {
		logger.Warnf("kubectl apply %s failed: %s", filePath, err.Error())
	}
	// check install csi success or not
	if err = utils.RetryFunc(ctx, opts, 10*time.Second, "checkCSIInstall", c.checkCSIHealth); err != nil {
		return nil, err
	}
	// delete pod, pvc, pv
	_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "delete", "-f", filePath)
	if err != nil {
		logger.Warnf("kubectl delete %s failed: %s", filePath, err.Error())
	}

	return nil, err
}

func (c *CSIHealthCheck) checkCSIHealth(ctx context.Context, opts component.Options) error {
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
	return fmt.Errorf("csi installation not completed")
}

func (c *CSIHealthCheck) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (c *CSIHealthCheck) NewInstance() component.ObjectMeta {
	return &CSIHealthCheck{}
}

// GetCheckCSIHealthStep get the common step
func (c *CSIHealthCheck) GetCheckCSIHealthStep(nodes []v1.StepNode, storageClassName string) ([]v1.Step, error) {
	steps := make([]v1.Step, 0, 2)
	c.StorageClassName = storageClassName
	aData, err := json.Marshal(c)
	if err != nil {
		return steps, err
	}
	steps = append(steps, []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "renderCheckCSIHealthManifests",
			Timeout:    metav1.Duration{Duration: 3 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type: v1.CommandTemplateRender,
					Template: &v1.TemplateCommand{
						Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, CsiHealthCheck, version, HealthCheck),
						Data:     aData,
					},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "checkCSIHealth",
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, CsiHealthCheck, version, HealthCheck),
					CustomCommand: aData,
				},
			},
		},
	}...)

	return steps, nil
}

func (c *CSIHealthCheck) renderCSIHealthCheckPVC(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, CSIHealthCheckTemplate, c)
	return err
}

func (c *CSIHealthCheck) Render(ctx context.Context, opts component.Options) error {
	if err := os.MkdirAll(ManifestsDir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(ManifestsDir, checkCSIHealthFile)
	return fileutil.WriteFileWithContext(ctx, filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		c.renderCSIHealthCheckPVC, opts.DryRun)
}
