package operationv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

const CommandStepExecutorName = "CommandStep/v1"

type CommandStepPayload struct {
	Step          corev1.Step `json:"step"`
	LastTaskReply string      `json:"lastTaskReply,omitempty"`
	DryRun        bool        `json:"dryRun,omitempty"`
}

// CommandStepExecutor runs the existing versioned KubeClipper Step model. It
// is registered only by kc-agent; clients cannot use it as an online shell API.
type CommandStepExecutor struct {
	OpLog      component.OperationLogFile
	RepoMirror string
}

func (e CommandStepExecutor) Reconcile(ctx context.Context, task *operations.OperationTask, log io.Writer) (operations.TaskResult, error) {
	var payload CommandStepPayload
	decoder := json.NewDecoder(bytes.NewReader(task.Spec.Payload.Raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return operations.TaskResult{}, fmt.Errorf("decode CommandStep/v1 payload: %w", err)
	}
	ctx = component.WithOperationID(ctx, string(task.UID))
	ctx = component.WithStepID(ctx, "task")
	ctx = component.WithOplog(ctx, e.OpLog)
	ctx = component.WithRepoMirror(ctx, e.RepoMirror)
	if payload.LastTaskReply != "" {
		ctx = component.WithExtraData(ctx, []byte(payload.LastTaskReply))
	}

	commands := make([]corev1.Command, 0, len(payload.Step.BeforeRunCommands)+len(payload.Step.Commands)+len(payload.Step.AfterRunCommands))
	commands = append(commands, payload.Step.BeforeRunCommands...)
	commands = append(commands, payload.Step.Commands...)
	commands = append(commands, payload.Step.AfterRunCommands...)
	var response []byte
	for index := range commands {
		command := &commands[index]
		if err := ctx.Err(); err != nil {
			return operations.TaskResult{}, err
		}
		_, _ = fmt.Fprintf(log, "command %d/%d: %s\n", index+1, len(commands), command.Type)
		switch command.Type {
		case corev1.CommandShell:
			if len(command.ShellCommand) == 0 {
				return operations.TaskResult{}, fmt.Errorf("shell command must not be empty")
			}
			if _, err := cmdutil.RunCmdWithContext(ctx, payload.DryRun, command.ShellCommand[0], command.ShellCommand[1:]...); err != nil {
				return operations.TaskResult{}, fmt.Errorf("run shell command: %w", err)
			}
		case corev1.CommandTemplateRender:
			if err := e.renderTemplate(ctx, command.Template, payload.DryRun); err != nil {
				return operations.TaskResult{}, err
			}
		case corev1.CommandCustom:
			value, err := e.runCustom(ctx, &payload.Step, command, payload.DryRun)
			if err != nil {
				return operations.TaskResult{}, err
			}
			response = value
		default:
			return operations.TaskResult{}, fmt.Errorf("unsupported command type %q", command.Type)
		}
	}
	if len(response) > operations.MaxOutputValueSize {
		return operations.TaskResult{}, fmt.Errorf("step response exceeds %d bytes", operations.MaxOutputValueSize)
	}
	return operations.TaskResult{Outputs: map[string]string{"response": string(response)}}, nil
}

func (CommandStepExecutor) renderTemplate(ctx context.Context, command *corev1.TemplateCommand, dryRun bool) error {
	if command == nil {
		return fmt.Errorf("template command is required")
	}
	registered, ok := component.LoadTemplate(command.Identity)
	if !ok {
		return fmt.Errorf("template renderer %q is not registered", command.Identity)
	}
	instance := registered.NewInstance()
	renderer, ok := instance.(component.TemplateRender)
	if !ok {
		return fmt.Errorf("template renderer %q has invalid type", command.Identity)
	}
	if err := json.Unmarshal(command.Data, renderer); err != nil {
		return fmt.Errorf("decode template renderer %q: %w", command.Identity, err)
	}
	if err := renderer.Render(ctx, component.Options{DryRun: dryRun}); err != nil {
		return fmt.Errorf("render template %q: %w", command.Identity, err)
	}
	return nil
}

func (CommandStepExecutor) runCustom(ctx context.Context, step *corev1.Step, command *corev1.Command, dryRun bool) ([]byte, error) {
	registered, ok := component.LoadAgentStep(command.Identity)
	if !ok {
		return nil, fmt.Errorf("custom step %q is not registered", command.Identity)
	}
	instance := registered.NewInstance()
	if err := json.Unmarshal(command.CustomCommand, instance); err != nil {
		return nil, fmt.Errorf("decode custom step %q: %w", command.Identity, err)
	}
	runnable, ok := instance.(component.StepRunnable)
	if !ok {
		return nil, fmt.Errorf("custom step %q has invalid type", command.Identity)
	}
	if step.Action == corev1.ActionInstall {
		return runnable.Install(ctx, component.Options{DryRun: dryRun})
	}
	if step.Action == corev1.ActionUninstall {
		return runnable.Uninstall(ctx, component.Options{DryRun: dryRun})
	}
	return nil, fmt.Errorf("custom step %q has unsupported action %q", command.Identity, step.Action)
}
