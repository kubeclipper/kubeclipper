package framework

import "time"

var (
	// Default timeouts to be used in TimeoutContext
	clusterInstall      = 20 * time.Minute
	clusterInstallShort = 5 * time.Minute
	clusterDelete       = 10 * time.Minute
	commonTimeout       = 5 * time.Minute
)

// TimeoutContext contains timeout settings for several actions.
type TimeoutContext struct {
	// ClusterInstall is how long to wait for the pod to be started.
	// Use it in create ha cluster case
	ClusterInstall time.Duration

	// ClusterInstallShort is same as `ClusterInstall`, but shorter.
	// Use it in create aio cluster case
	ClusterInstallShort time.Duration

	// ClusterDelete is how long to wait for the cluster to be deleted.
	ClusterDelete time.Duration

	CommonTimeout time.Duration
}

// NewTimeoutContextWithDefaults returns a TimeoutContext with default values.
func NewTimeoutContextWithDefaults() *TimeoutContext {
	return &TimeoutContext{
		ClusterInstall:      clusterInstall,
		ClusterInstallShort: clusterInstallShort,
		ClusterDelete:       clusterDelete,
		CommonTimeout:       commonTimeout,
	}
}
