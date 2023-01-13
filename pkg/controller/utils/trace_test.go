package utils

import (
	"reflect"
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func TestTrace(t *testing.T) {
	controller := "controller1"
	log := logger.WithName("test")
	type args struct {
		log        logger.Logging
		controller string
	}
	tests := []struct {
		name string
		args args
		want func()
	}{
		{
			name: "base",
			args: args{
				log:        log,
				controller: controller,
			},
			want: func() {
				start := time.Now()
				log.Debugf("exit %s ,spend:%v", controller, time.Since(start))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Trace(tt.args.log, tt.args.controller); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Trace() = %v, want %v", got, tt.want)
			}
		})
	}
}
