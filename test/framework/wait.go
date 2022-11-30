package framework

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	apierror "github.com/kubeclipper/kubeclipper/pkg/errors"
)

const (
	// Poll is how often to poll clusters.
	Poll = 15 * time.Second
)

type timeoutError struct {
	msg             string
	observedObjects []interface{}
}

func (e *timeoutError) Error() string {
	return e.msg
}

func TimeoutError(msg string, observedObjects ...interface{}) error {
	return &timeoutError{
		msg:             msg,
		observedObjects: observedObjects,
	}
}

func IsTimeout(err error) bool {
	if err == wait.ErrWaitTimeout {
		return true
	}
	if _, ok := err.(*timeoutError); ok {
		return true
	}
	return false
}

// MaybeTimeoutError returns a TimeoutError if err is a timeout. Otherwise, wrap err.
// taskFormat and taskArgs should be the task being performed when the error occurred,
// e.g. "waiting for pod to be running".
func MaybeTimeoutError(err error, taskFormat string, taskArgs ...interface{}) error {
	if IsTimeout(err) {
		return TimeoutError(fmt.Sprintf("timed out while "+taskFormat, taskArgs...))
	} else if err != nil {
		return fmt.Errorf("error while %s: %w", fmt.Sprintf(taskFormat, taskArgs...), err)
	} else {
		return nil
	}
}

// HandleWaitingAPIError handles an error from an API request in the context of a Wait function.
// If the error is retryable, sleep the recommended delay and ignore the error.
// If the erorr is terminal, return it.
func HandleWaitingAPIError(err error, retryNotFound bool, taskFormat string, taskArgs ...interface{}) (bool, error) {
	taskDescription := fmt.Sprintf(taskFormat, taskArgs...)
	if retryNotFound && apierror.IsNotFound(err) {
		Logf("Ignoring NotFound error while " + taskDescription)
		return false, nil
	}
	if retry, delay := shouldRetry(err); retry {
		Logf("Retryable error while %s, retrying after %v: %v", taskDescription, delay, err)
		if delay > 0 {
			time.Sleep(delay)
		}
		return false, nil
	}
	Logf("Encountered non-retryable error while %s: %v", taskDescription, err)
	return false, err
}

// Decide whether to retry an API request. Optionally include a delay to retry after.
func shouldRetry(err error) (retry bool, retryAfter time.Duration) {
	// these errors indicate a transient error that should be retried.
	if apierror.IsInternalError(err) || apierror.IsTooManyRequests(err) {
		return true, 0
	}

	return false, 0
}
