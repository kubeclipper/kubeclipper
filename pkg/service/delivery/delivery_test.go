/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package delivery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/service"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func getOperationCase1() *v1.Operation {
	return &v1.Operation{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Steps: []v1.Step{
			{
				ID:         "1",
				Name:       "step-1",
				Timeout:    metav1.Duration{Duration: 3 * time.Second},
				ErrIgnore:  false,
				RetryTimes: 1,
				Nodes: []v1.StepNode{{
					ID:       "node1",
					IPv4:     "127.0.0.1",
					Hostname: "virtual",
				}},
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"/bin/sh", "-c", "echo", "test"},
					},
				},
			},
			{
				ID:         "2",
				Name:       "step-2",
				Timeout:    metav1.Duration{Duration: 3 * time.Second},
				ErrIgnore:  false,
				RetryTimes: 1,
				Nodes: []v1.StepNode{
					{
						ID:       "node1",
						IPv4:     "192.168.10.10",
						Hostname: "node1",
					},
					{
						ID:       "node2",
						IPv4:     "192.168.10.11",
						Hostname: "node2",
					},
				},
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"/bin/sh", "-c", "echo", "test2"},
					},
				},
			},
		},
		Status: v1.OperationStatus{},
	}
}

func TestService_DeliverTaskOperation(t *testing.T) {
	//mockCtl := gomock.NewController(t)
	//defer mockCtl.Finish()
	//mockNatsio := mock_natsio.NewMockInterface(mockCtl)
	//mockClusterOp := mock_cluster.NewMockOperator(mockCtl)
	//mockOperation := mock_operation.NewMockOperator(mockCtl)
	//mockLease := mock_lease.NewMockOperator(mockCtl)
	//
	//s := Service{
	//	nodeReportSubject: "node1",
	//	queueGroup:        "kubeclipper-server",
	//	subjectSuffix:     "kubeclipper-server",
	//	client:            mockNatsio,
	//	clusterOperator:   mockClusterOp,
	//	leaseOperator:     mockLease,
	//	opOperator:        mockOperation,
	//	lock:              nil,
	//	leaderElector:     nil,
	//	stepStatusChan:    make(chan stepStatus, 256),
	//}
	//
	//opCase1 := getOperationCase1()
	//case1Step1Bytes := mustJSONMarshal(opCase1.Steps[0])
	//case1Step2Bytes := mustJSONMarshal(opCase1.Steps[1])
	//
	//case1Step1Payload := mustInitPayload(&opCase1.Steps[0])
	//case1Step2Payload := mustInitPayload(&opCase1.Steps[1])
	//
	//case1Step1Msg := &natsio.Msg{
	//	Subject: fmt.Sprintf(service.MsgSubjectFormat, opCase1.Steps[0].Nodes[0], s.subjectSuffix),
	//	From:    "",
	//	To:      "",
	//	Step:    "",
	//	Timeout: 0,
	//	Data:    case1Step1Payload,
	//}
	//
	//mockNatsio.EXPECT().Request(case1Step1Msg, nil).Return().DoAndReturn()
	//
	//if err := s.DeliverTaskOperation(context.TODO(), getOperationCase1()); err != nil {
	//	t.Error(err)
	//}

}

func mustJSONMarshal(v interface{}) []byte {
	if data, err := json.Marshal(v); err != nil {
		panic(err)
	} else {
		return data
	}
}

func mustInitPayload(step *v1.Step) []byte {
	if data, err := initPayload("", service.OperationRunTask, step, nil, nil, false, false); err != nil {
		panic(err)
	} else {
		return data
	}
}
