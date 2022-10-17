package cloudprovidercontroller

import (
	"strings"

	pkgerrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// NewCondition creates a new deployment condition.
func NewCondition(condType v1.CloudProviderConditionType, status v1.ConditionStatus, reason, message string) *v1.CloudProviderCondition {
	return &v1.CloudProviderCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// convert err to a human-readable msg
func convert(err error) string {
	if strings.Contains(err.Error(), "ssh: handshake failed") {
		return "ssh connect failed,please check your ssh user、password or privateKey"
	}

	return pkgerrors.Cause(err).Error()
}

// GetCondition returns the condition with the provided type.
func GetCondition(status v1.CloudProviderStatus, condType v1.CloudProviderConditionType) *v1.CloudProviderCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition updates the deployment to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetCondition(status *v1.CloudProviderStatus, condition v1.CloudProviderCondition) {
	currentCond := GetCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// RemoveCondition removes the deployment condition with the provided type.
func RemoveCondition(status *v1.CloudProviderStatus, condType v1.CloudProviderConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of deployment conditions without conditions with the provided type.
func filterOutCondition(conditions []v1.CloudProviderCondition, condType v1.CloudProviderConditionType) []v1.CloudProviderCondition {
	var newConditions []v1.CloudProviderCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
