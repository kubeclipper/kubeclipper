package utils

import (
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func Trace(log logger.Logging, controller string) func() {
	start := time.Now()
	log.Debugf("enter %s", controller)
	return func() {
		log.Debugf("exit %s ,spend:%v", controller, time.Since(start))
	}
}
