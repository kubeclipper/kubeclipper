/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package mock_cluster

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/kubeclipper/kubeclipper/pkg/models"
	query "github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	watch "k8s.io/apimachinery/pkg/watch"
)

// MockOperatorReader is a mock of OperatorReader interface.
type MockOperatorReader struct {
	ctrl     *gomock.Controller
	recorder *MockOperatorReaderMockRecorder
}

// MockOperatorReaderMockRecorder is the mock recorder for MockOperatorReader.
type MockOperatorReaderMockRecorder struct {
	mock *MockOperatorReader
}

// NewMockOperatorReader creates a new mock instance.
func NewMockOperatorReader(ctrl *gomock.Controller) *MockOperatorReader {
	mock := &MockOperatorReader{ctrl: ctrl}
	mock.recorder = &MockOperatorReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOperatorReader) EXPECT() *MockOperatorReaderMockRecorder {
	return m.recorder
}

// GetBackup mocks base method.
func (m *MockOperatorReader) GetBackup(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackup", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackup indicates an expected call of GetBackup.
func (mr *MockOperatorReaderMockRecorder) GetBackup(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackup", reflect.TypeOf((*MockOperatorReader)(nil).GetBackup), ctx, cluster, name)
}

// GetBackupEx mocks base method.
func (m *MockOperatorReader) GetBackupEx(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupEx indicates an expected call of GetBackupEx.
func (mr *MockOperatorReaderMockRecorder) GetBackupEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupEx", reflect.TypeOf((*MockOperatorReader)(nil).GetBackupEx), ctx, cluster, name)
}

// GetBackupPoint mocks base method.
func (m *MockOperatorReader) GetBackupPoint(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPoint", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPoint indicates an expected call of GetBackupPoint.
func (mr *MockOperatorReaderMockRecorder) GetBackupPoint(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPoint", reflect.TypeOf((*MockOperatorReader)(nil).GetBackupPoint), ctx, name, resourceVersion)
}

// GetBackupPointEx mocks base method.
func (m *MockOperatorReader) GetBackupPointEx(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPointEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPointEx indicates an expected call of GetBackupPointEx.
func (mr *MockOperatorReaderMockRecorder) GetBackupPointEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPointEx", reflect.TypeOf((*MockOperatorReader)(nil).GetBackupPointEx), ctx, name, resourceVersion)
}

// GetCluster mocks base method.
func (m *MockOperatorReader) GetCluster(ctx context.Context, name string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCluster", ctx, name)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCluster indicates an expected call of GetCluster.
func (mr *MockOperatorReaderMockRecorder) GetCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCluster", reflect.TypeOf((*MockOperatorReader)(nil).GetCluster), ctx, name)
}

// GetClusterEx mocks base method.
func (m *MockOperatorReader) GetClusterEx(ctx context.Context, name, resourceVersion string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterEx indicates an expected call of GetClusterEx.
func (mr *MockOperatorReaderMockRecorder) GetClusterEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterEx", reflect.TypeOf((*MockOperatorReader)(nil).GetClusterEx), ctx, name, resourceVersion)
}

// GetCronBackup mocks base method.
func (m *MockOperatorReader) GetCronBackup(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackup", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackup indicates an expected call of GetCronBackup.
func (mr *MockOperatorReaderMockRecorder) GetCronBackup(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackup", reflect.TypeOf((*MockOperatorReader)(nil).GetCronBackup), ctx, name, resourceVersion)
}

// GetCronBackupEx mocks base method.
func (m *MockOperatorReader) GetCronBackupEx(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackupEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackupEx indicates an expected call of GetCronBackupEx.
func (mr *MockOperatorReaderMockRecorder) GetCronBackupEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackupEx", reflect.TypeOf((*MockOperatorReader)(nil).GetCronBackupEx), ctx, name, resourceVersion)
}

// GetDomain mocks base method.
func (m *MockOperatorReader) GetDomain(ctx context.Context, name string) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomain", ctx, name)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomain indicates an expected call of GetDomain.
func (mr *MockOperatorReaderMockRecorder) GetDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomain", reflect.TypeOf((*MockOperatorReader)(nil).GetDomain), ctx, name)
}

// GetNode mocks base method.
func (m *MockOperatorReader) GetNode(ctx context.Context, name string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNode", ctx, name)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNode indicates an expected call of GetNode.
func (mr *MockOperatorReaderMockRecorder) GetNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNode", reflect.TypeOf((*MockOperatorReader)(nil).GetNode), ctx, name)
}

// GetNodeEx mocks base method.
func (m *MockOperatorReader) GetNodeEx(ctx context.Context, name, resourceVersion string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeEx indicates an expected call of GetNodeEx.
func (mr *MockOperatorReaderMockRecorder) GetNodeEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeEx", reflect.TypeOf((*MockOperatorReader)(nil).GetNodeEx), ctx, name, resourceVersion)
}

// GetRegion mocks base method.
func (m *MockOperatorReader) GetRegion(ctx context.Context, name string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegion", ctx, name)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegion indicates an expected call of GetRegion.
func (mr *MockOperatorReaderMockRecorder) GetRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegion", reflect.TypeOf((*MockOperatorReader)(nil).GetRegion), ctx, name)
}

// GetRegionEx mocks base method.
func (m *MockOperatorReader) GetRegionEx(ctx context.Context, name, resourceVersion string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegionEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegionEx indicates an expected call of GetRegionEx.
func (mr *MockOperatorReaderMockRecorder) GetRegionEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegionEx", reflect.TypeOf((*MockOperatorReader)(nil).GetRegionEx), ctx, name, resourceVersion)
}

// ListBackupEx mocks base method.
func (m *MockOperatorReader) ListBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupEx indicates an expected call of ListBackupEx.
func (mr *MockOperatorReaderMockRecorder) ListBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupEx", reflect.TypeOf((*MockOperatorReader)(nil).ListBackupEx), ctx, query)
}

// ListBackupPointEx mocks base method.
func (m *MockOperatorReader) ListBackupPointEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPointEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPointEx indicates an expected call of ListBackupPointEx.
func (mr *MockOperatorReaderMockRecorder) ListBackupPointEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPointEx", reflect.TypeOf((*MockOperatorReader)(nil).ListBackupPointEx), ctx, query)
}

// ListBackupPoints mocks base method.
func (m *MockOperatorReader) ListBackupPoints(ctx context.Context, query *query.Query) (*v1.BackupPointList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPoints", ctx, query)
	ret0, _ := ret[0].(*v1.BackupPointList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPoints indicates an expected call of ListBackupPoints.
func (mr *MockOperatorReaderMockRecorder) ListBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPoints", reflect.TypeOf((*MockOperatorReader)(nil).ListBackupPoints), ctx, query)
}

// ListBackups mocks base method.
func (m *MockOperatorReader) ListBackups(ctx context.Context, query *query.Query) (*v1.BackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackups", ctx, query)
	ret0, _ := ret[0].(*v1.BackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackups indicates an expected call of ListBackups.
func (mr *MockOperatorReaderMockRecorder) ListBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackups", reflect.TypeOf((*MockOperatorReader)(nil).ListBackups), ctx, query)
}

// ListClusterEx mocks base method.
func (m *MockOperatorReader) ListClusterEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusterEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusterEx indicates an expected call of ListClusterEx.
func (mr *MockOperatorReaderMockRecorder) ListClusterEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusterEx", reflect.TypeOf((*MockOperatorReader)(nil).ListClusterEx), ctx, query)
}

// ListClusters mocks base method.
func (m *MockOperatorReader) ListClusters(ctx context.Context, query *query.Query) (*v1.ClusterList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusters", ctx, query)
	ret0, _ := ret[0].(*v1.ClusterList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusters indicates an expected call of ListClusters.
func (mr *MockOperatorReaderMockRecorder) ListClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusters", reflect.TypeOf((*MockOperatorReader)(nil).ListClusters), ctx, query)
}

// ListCronBackupEx mocks base method.
func (m *MockOperatorReader) ListCronBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackupEx indicates an expected call of ListCronBackupEx.
func (mr *MockOperatorReaderMockRecorder) ListCronBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackupEx", reflect.TypeOf((*MockOperatorReader)(nil).ListCronBackupEx), ctx, query)
}

// ListCronBackups mocks base method.
func (m *MockOperatorReader) ListCronBackups(ctx context.Context, query *query.Query) (*v1.CronBackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackups", ctx, query)
	ret0, _ := ret[0].(*v1.CronBackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackups indicates an expected call of ListCronBackups.
func (mr *MockOperatorReaderMockRecorder) ListCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackups", reflect.TypeOf((*MockOperatorReader)(nil).ListCronBackups), ctx, query)
}

// ListDomains mocks base method.
func (m *MockOperatorReader) ListDomains(ctx context.Context, query *query.Query) (*v1.DomainList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomains", ctx, query)
	ret0, _ := ret[0].(*v1.DomainList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomains indicates an expected call of ListDomains.
func (mr *MockOperatorReaderMockRecorder) ListDomains(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomains", reflect.TypeOf((*MockOperatorReader)(nil).ListDomains), ctx, query)
}

// ListDomainsEx mocks base method.
func (m *MockOperatorReader) ListDomainsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomainsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomainsEx indicates an expected call of ListDomainsEx.
func (mr *MockOperatorReaderMockRecorder) ListDomainsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomainsEx", reflect.TypeOf((*MockOperatorReader)(nil).ListDomainsEx), ctx, query)
}

// ListNodes mocks base method.
func (m *MockOperatorReader) ListNodes(ctx context.Context, query *query.Query) (*v1.NodeList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodes", ctx, query)
	ret0, _ := ret[0].(*v1.NodeList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodes indicates an expected call of ListNodes.
func (mr *MockOperatorReaderMockRecorder) ListNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodes", reflect.TypeOf((*MockOperatorReader)(nil).ListNodes), ctx, query)
}

// ListNodesEx mocks base method.
func (m *MockOperatorReader) ListNodesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodesEx indicates an expected call of ListNodesEx.
func (mr *MockOperatorReaderMockRecorder) ListNodesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodesEx", reflect.TypeOf((*MockOperatorReader)(nil).ListNodesEx), ctx, query)
}

// ListRecordsEx mocks base method.
func (m *MockOperatorReader) ListRecordsEx(ctx context.Context, name, subdomain string, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecordsEx", ctx, name, subdomain, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecordsEx indicates an expected call of ListRecordsEx.
func (mr *MockOperatorReaderMockRecorder) ListRecordsEx(ctx, name, subdomain, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecordsEx", reflect.TypeOf((*MockOperatorReader)(nil).ListRecordsEx), ctx, name, subdomain, query)
}

// ListRegionEx mocks base method.
func (m *MockOperatorReader) ListRegionEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegionEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegionEx indicates an expected call of ListRegionEx.
func (mr *MockOperatorReaderMockRecorder) ListRegionEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegionEx", reflect.TypeOf((*MockOperatorReader)(nil).ListRegionEx), ctx, query)
}

// ListRegions mocks base method.
func (m *MockOperatorReader) ListRegions(ctx context.Context, query *query.Query) (*v1.RegionList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegions", ctx, query)
	ret0, _ := ret[0].(*v1.RegionList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegions indicates an expected call of ListRegions.
func (mr *MockOperatorReaderMockRecorder) ListRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegions", reflect.TypeOf((*MockOperatorReader)(nil).ListRegions), ctx, query)
}

// WatchBackupPoints mocks base method.
func (m *MockOperatorReader) WatchBackupPoints(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackupPoints", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackupPoints indicates an expected call of WatchBackupPoints.
func (mr *MockOperatorReaderMockRecorder) WatchBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackupPoints", reflect.TypeOf((*MockOperatorReader)(nil).WatchBackupPoints), ctx, query)
}

// WatchBackups mocks base method.
func (m *MockOperatorReader) WatchBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackups indicates an expected call of WatchBackups.
func (mr *MockOperatorReaderMockRecorder) WatchBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackups", reflect.TypeOf((*MockOperatorReader)(nil).WatchBackups), ctx, query)
}

// WatchClusters mocks base method.
func (m *MockOperatorReader) WatchClusters(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchClusters", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchClusters indicates an expected call of WatchClusters.
func (mr *MockOperatorReaderMockRecorder) WatchClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchClusters", reflect.TypeOf((*MockOperatorReader)(nil).WatchClusters), ctx, query)
}

// WatchCronBackups mocks base method.
func (m *MockOperatorReader) WatchCronBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCronBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCronBackups indicates an expected call of WatchCronBackups.
func (mr *MockOperatorReaderMockRecorder) WatchCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCronBackups", reflect.TypeOf((*MockOperatorReader)(nil).WatchCronBackups), ctx, query)
}

// WatchDomain mocks base method.
func (m *MockOperatorReader) WatchDomain(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchDomain", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchDomain indicates an expected call of WatchDomain.
func (mr *MockOperatorReaderMockRecorder) WatchDomain(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchDomain", reflect.TypeOf((*MockOperatorReader)(nil).WatchDomain), ctx, query)
}

// WatchNodes mocks base method.
func (m *MockOperatorReader) WatchNodes(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchNodes", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchNodes indicates an expected call of WatchNodes.
func (mr *MockOperatorReaderMockRecorder) WatchNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchNodes", reflect.TypeOf((*MockOperatorReader)(nil).WatchNodes), ctx, query)
}

// WatchRegions mocks base method.
func (m *MockOperatorReader) WatchRegions(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRegions", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRegions indicates an expected call of WatchRegions.
func (mr *MockOperatorReaderMockRecorder) WatchRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRegions", reflect.TypeOf((*MockOperatorReader)(nil).WatchRegions), ctx, query)
}

// MockOperatorWriter is a mock of OperatorWriter interface.
type MockOperatorWriter struct {
	ctrl     *gomock.Controller
	recorder *MockOperatorWriterMockRecorder
}

// MockOperatorWriterMockRecorder is the mock recorder for MockOperatorWriter.
type MockOperatorWriterMockRecorder struct {
	mock *MockOperatorWriter
}

// NewMockOperatorWriter creates a new mock instance.
func NewMockOperatorWriter(ctrl *gomock.Controller) *MockOperatorWriter {
	mock := &MockOperatorWriter{ctrl: ctrl}
	mock.recorder = &MockOperatorWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOperatorWriter) EXPECT() *MockOperatorWriterMockRecorder {
	return m.recorder
}

// CreateBackup mocks base method.
func (m *MockOperatorWriter) CreateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackup indicates an expected call of CreateBackup.
func (mr *MockOperatorWriterMockRecorder) CreateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackup", reflect.TypeOf((*MockOperatorWriter)(nil).CreateBackup), ctx, backup)
}

// CreateBackupPoint mocks base method.
func (m *MockOperatorWriter) CreateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackupPoint indicates an expected call of CreateBackupPoint.
func (mr *MockOperatorWriterMockRecorder) CreateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackupPoint", reflect.TypeOf((*MockOperatorWriter)(nil).CreateBackupPoint), ctx, backupPoint)
}

// CreateCluster mocks base method.
func (m *MockOperatorWriter) CreateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCluster indicates an expected call of CreateCluster.
func (mr *MockOperatorWriterMockRecorder) CreateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCluster", reflect.TypeOf((*MockOperatorWriter)(nil).CreateCluster), ctx, cluster)
}

// CreateCronBackup mocks base method.
func (m *MockOperatorWriter) CreateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCronBackup indicates an expected call of CreateCronBackup.
func (mr *MockOperatorWriterMockRecorder) CreateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCronBackup", reflect.TypeOf((*MockOperatorWriter)(nil).CreateCronBackup), ctx, cronBackup)
}

// CreateDomain mocks base method.
func (m *MockOperatorWriter) CreateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDomain indicates an expected call of CreateDomain.
func (mr *MockOperatorWriterMockRecorder) CreateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDomain", reflect.TypeOf((*MockOperatorWriter)(nil).CreateDomain), ctx, domain)
}

// CreateNode mocks base method.
func (m *MockOperatorWriter) CreateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNode indicates an expected call of CreateNode.
func (mr *MockOperatorWriterMockRecorder) CreateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNode", reflect.TypeOf((*MockOperatorWriter)(nil).CreateNode), ctx, node)
}

// CreateRegion mocks base method.
func (m *MockOperatorWriter) CreateRegion(ctx context.Context, region *v1.Region) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRegion", ctx, region)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRegion indicates an expected call of CreateRegion.
func (mr *MockOperatorWriterMockRecorder) CreateRegion(ctx, region interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRegion", reflect.TypeOf((*MockOperatorWriter)(nil).CreateRegion), ctx, region)
}

// DeleteBackup mocks base method.
func (m *MockOperatorWriter) DeleteBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackup indicates an expected call of DeleteBackup.
func (mr *MockOperatorWriterMockRecorder) DeleteBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackup", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteBackup), ctx, name)
}

// DeleteBackupPoint mocks base method.
func (m *MockOperatorWriter) DeleteBackupPoint(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackupPoint", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackupPoint indicates an expected call of DeleteBackupPoint.
func (mr *MockOperatorWriterMockRecorder) DeleteBackupPoint(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackupPoint", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteBackupPoint), ctx, name)
}

// DeleteCluster mocks base method.
func (m *MockOperatorWriter) DeleteCluster(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCluster", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCluster indicates an expected call of DeleteCluster.
func (mr *MockOperatorWriterMockRecorder) DeleteCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCluster", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteCluster), ctx, name)
}

// DeleteCronBackup mocks base method.
func (m *MockOperatorWriter) DeleteCronBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackup indicates an expected call of DeleteCronBackup.
func (mr *MockOperatorWriterMockRecorder) DeleteCronBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackup", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteCronBackup), ctx, name)
}

// DeleteCronBackupCollection mocks base method.
func (m *MockOperatorWriter) DeleteCronBackupCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackupCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackupCollection indicates an expected call of DeleteCronBackupCollection.
func (mr *MockOperatorWriterMockRecorder) DeleteCronBackupCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackupCollection", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteCronBackupCollection), ctx, query)
}

// DeleteDomain mocks base method.
func (m *MockOperatorWriter) DeleteDomain(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDomain", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDomain indicates an expected call of DeleteDomain.
func (mr *MockOperatorWriterMockRecorder) DeleteDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDomain", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteDomain), ctx, name)
}

// DeleteNode mocks base method.
func (m *MockOperatorWriter) DeleteNode(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNode", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNode indicates an expected call of DeleteNode.
func (mr *MockOperatorWriterMockRecorder) DeleteNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNode", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteNode), ctx, name)
}

// DeleteRegion mocks base method.
func (m *MockOperatorWriter) DeleteRegion(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRegion", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRegion indicates an expected call of DeleteRegion.
func (mr *MockOperatorWriterMockRecorder) DeleteRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRegion", reflect.TypeOf((*MockOperatorWriter)(nil).DeleteRegion), ctx, name)
}

// UpdateBackup mocks base method.
func (m *MockOperatorWriter) UpdateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackup indicates an expected call of UpdateBackup.
func (mr *MockOperatorWriterMockRecorder) UpdateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackup", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateBackup), ctx, backup)
}

// UpdateBackupPoint mocks base method.
func (m *MockOperatorWriter) UpdateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackupPoint indicates an expected call of UpdateBackupPoint.
func (mr *MockOperatorWriterMockRecorder) UpdateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackupPoint", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateBackupPoint), ctx, backupPoint)
}

// UpdateCluster mocks base method.
func (m *MockOperatorWriter) UpdateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCluster indicates an expected call of UpdateCluster.
func (mr *MockOperatorWriterMockRecorder) UpdateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCluster", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateCluster), ctx, cluster)
}

// UpdateCronBackup mocks base method.
func (m *MockOperatorWriter) UpdateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCronBackup indicates an expected call of UpdateCronBackup.
func (mr *MockOperatorWriterMockRecorder) UpdateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCronBackup", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateCronBackup), ctx, cronBackup)
}

// UpdateDomain mocks base method.
func (m *MockOperatorWriter) UpdateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDomain indicates an expected call of UpdateDomain.
func (mr *MockOperatorWriterMockRecorder) UpdateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDomain", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateDomain), ctx, domain)
}

// UpdateNode mocks base method.
func (m *MockOperatorWriter) UpdateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateNode indicates an expected call of UpdateNode.
func (mr *MockOperatorWriterMockRecorder) UpdateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNode", reflect.TypeOf((*MockOperatorWriter)(nil).UpdateNode), ctx, node)
}

// MockOperator is a mock of Operator interface.
type MockOperator struct {
	ctrl     *gomock.Controller
	recorder *MockOperatorMockRecorder
}

// MockOperatorMockRecorder is the mock recorder for MockOperator.
type MockOperatorMockRecorder struct {
	mock *MockOperator
}

// NewMockOperator creates a new mock instance.
func NewMockOperator(ctrl *gomock.Controller) *MockOperator {
	mock := &MockOperator{ctrl: ctrl}
	mock.recorder = &MockOperatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOperator) EXPECT() *MockOperatorMockRecorder {
	return m.recorder
}

// CreateBackup mocks base method.
func (m *MockOperator) CreateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackup indicates an expected call of CreateBackup.
func (mr *MockOperatorMockRecorder) CreateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackup", reflect.TypeOf((*MockOperator)(nil).CreateBackup), ctx, backup)
}

// CreateBackupPoint mocks base method.
func (m *MockOperator) CreateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackupPoint indicates an expected call of CreateBackupPoint.
func (mr *MockOperatorMockRecorder) CreateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackupPoint", reflect.TypeOf((*MockOperator)(nil).CreateBackupPoint), ctx, backupPoint)
}

// CreateCloudProvider mocks base method.
func (m *MockOperator) CreateCloudProvider(ctx context.Context, cp *v1.CloudProvider) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCloudProvider", ctx, cp)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCloudProvider indicates an expected call of CreateCloudProvider.
func (mr *MockOperatorMockRecorder) CreateCloudProvider(ctx, cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCloudProvider", reflect.TypeOf((*MockOperator)(nil).CreateCloudProvider), ctx, cp)
}

// CreateCluster mocks base method.
func (m *MockOperator) CreateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCluster indicates an expected call of CreateCluster.
func (mr *MockOperatorMockRecorder) CreateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCluster", reflect.TypeOf((*MockOperator)(nil).CreateCluster), ctx, cluster)
}

// CreateCronBackup mocks base method.
func (m *MockOperator) CreateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCronBackup indicates an expected call of CreateCronBackup.
func (mr *MockOperatorMockRecorder) CreateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCronBackup", reflect.TypeOf((*MockOperator)(nil).CreateCronBackup), ctx, cronBackup)
}

// CreateDomain mocks base method.
func (m *MockOperator) CreateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDomain indicates an expected call of CreateDomain.
func (mr *MockOperatorMockRecorder) CreateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDomain", reflect.TypeOf((*MockOperator)(nil).CreateDomain), ctx, domain)
}

// CreateNode mocks base method.
func (m *MockOperator) CreateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNode indicates an expected call of CreateNode.
func (mr *MockOperatorMockRecorder) CreateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNode", reflect.TypeOf((*MockOperator)(nil).CreateNode), ctx, node)
}

// CreateRegion mocks base method.
func (m *MockOperator) CreateRegion(ctx context.Context, region *v1.Region) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRegion", ctx, region)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRegion indicates an expected call of CreateRegion.
func (mr *MockOperatorMockRecorder) CreateRegion(ctx, region interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRegion", reflect.TypeOf((*MockOperator)(nil).CreateRegion), ctx, region)
}

// CreateRegistry mocks base method.
func (m *MockOperator) CreateRegistry(ctx context.Context, r *v1.Registry) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRegistry", ctx, r)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRegistry indicates an expected call of CreateRegistry.
func (mr *MockOperatorMockRecorder) CreateRegistry(ctx, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRegistry", reflect.TypeOf((*MockOperator)(nil).CreateRegistry), ctx, r)
}

// CreateTemplate mocks base method.
func (m *MockOperator) CreateTemplate(ctx context.Context, template *v1.Template) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTemplate", ctx, template)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTemplate indicates an expected call of CreateTemplate.
func (mr *MockOperatorMockRecorder) CreateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTemplate", reflect.TypeOf((*MockOperator)(nil).CreateTemplate), ctx, template)
}

// DeleteBackup mocks base method.
func (m *MockOperator) DeleteBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackup indicates an expected call of DeleteBackup.
func (mr *MockOperatorMockRecorder) DeleteBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackup", reflect.TypeOf((*MockOperator)(nil).DeleteBackup), ctx, name)
}

// DeleteBackupPoint mocks base method.
func (m *MockOperator) DeleteBackupPoint(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackupPoint", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackupPoint indicates an expected call of DeleteBackupPoint.
func (mr *MockOperatorMockRecorder) DeleteBackupPoint(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackupPoint", reflect.TypeOf((*MockOperator)(nil).DeleteBackupPoint), ctx, name)
}

// DeleteCloudProvider mocks base method.
func (m *MockOperator) DeleteCloudProvider(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCloudProvider", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCloudProvider indicates an expected call of DeleteCloudProvider.
func (mr *MockOperatorMockRecorder) DeleteCloudProvider(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCloudProvider", reflect.TypeOf((*MockOperator)(nil).DeleteCloudProvider), ctx, name)
}

// DeleteCluster mocks base method.
func (m *MockOperator) DeleteCluster(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCluster", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCluster indicates an expected call of DeleteCluster.
func (mr *MockOperatorMockRecorder) DeleteCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCluster", reflect.TypeOf((*MockOperator)(nil).DeleteCluster), ctx, name)
}

// DeleteCronBackup mocks base method.
func (m *MockOperator) DeleteCronBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackup indicates an expected call of DeleteCronBackup.
func (mr *MockOperatorMockRecorder) DeleteCronBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackup", reflect.TypeOf((*MockOperator)(nil).DeleteCronBackup), ctx, name)
}

// DeleteCronBackupCollection mocks base method.
func (m *MockOperator) DeleteCronBackupCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackupCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackupCollection indicates an expected call of DeleteCronBackupCollection.
func (mr *MockOperatorMockRecorder) DeleteCronBackupCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackupCollection", reflect.TypeOf((*MockOperator)(nil).DeleteCronBackupCollection), ctx, query)
}

// DeleteDomain mocks base method.
func (m *MockOperator) DeleteDomain(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDomain", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDomain indicates an expected call of DeleteDomain.
func (mr *MockOperatorMockRecorder) DeleteDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDomain", reflect.TypeOf((*MockOperator)(nil).DeleteDomain), ctx, name)
}

// DeleteNode mocks base method.
func (m *MockOperator) DeleteNode(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNode", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNode indicates an expected call of DeleteNode.
func (mr *MockOperatorMockRecorder) DeleteNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNode", reflect.TypeOf((*MockOperator)(nil).DeleteNode), ctx, name)
}

// DeleteRegion mocks base method.
func (m *MockOperator) DeleteRegion(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRegion", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRegion indicates an expected call of DeleteRegion.
func (mr *MockOperatorMockRecorder) DeleteRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRegion", reflect.TypeOf((*MockOperator)(nil).DeleteRegion), ctx, name)
}

// DeleteRegistry mocks base method.
func (m *MockOperator) DeleteRegistry(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRegistry", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRegistry indicates an expected call of DeleteRegistry.
func (mr *MockOperatorMockRecorder) DeleteRegistry(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRegistry", reflect.TypeOf((*MockOperator)(nil).DeleteRegistry), ctx, name)
}

// DeleteTemplate mocks base method.
func (m *MockOperator) DeleteTemplate(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTemplate", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTemplate indicates an expected call of DeleteTemplate.
func (mr *MockOperatorMockRecorder) DeleteTemplate(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTemplate", reflect.TypeOf((*MockOperator)(nil).DeleteTemplate), ctx, name)
}

// DeleteTemplateCollection mocks base method.
func (m *MockOperator) DeleteTemplateCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTemplateCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTemplateCollection indicates an expected call of DeleteTemplateCollection.
func (mr *MockOperatorMockRecorder) DeleteTemplateCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTemplateCollection", reflect.TypeOf((*MockOperator)(nil).DeleteTemplateCollection), ctx, query)
}

// GetBackup mocks base method.
func (m *MockOperator) GetBackup(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackup", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackup indicates an expected call of GetBackup.
func (mr *MockOperatorMockRecorder) GetBackup(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackup", reflect.TypeOf((*MockOperator)(nil).GetBackup), ctx, cluster, name)
}

// GetBackupEx mocks base method.
func (m *MockOperator) GetBackupEx(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupEx indicates an expected call of GetBackupEx.
func (mr *MockOperatorMockRecorder) GetBackupEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupEx", reflect.TypeOf((*MockOperator)(nil).GetBackupEx), ctx, cluster, name)
}

// GetBackupPoint mocks base method.
func (m *MockOperator) GetBackupPoint(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPoint", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPoint indicates an expected call of GetBackupPoint.
func (mr *MockOperatorMockRecorder) GetBackupPoint(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPoint", reflect.TypeOf((*MockOperator)(nil).GetBackupPoint), ctx, name, resourceVersion)
}

// GetBackupPointEx mocks base method.
func (m *MockOperator) GetBackupPointEx(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPointEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPointEx indicates an expected call of GetBackupPointEx.
func (mr *MockOperatorMockRecorder) GetBackupPointEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPointEx", reflect.TypeOf((*MockOperator)(nil).GetBackupPointEx), ctx, name, resourceVersion)
}

// GetCloudProvider mocks base method.
func (m *MockOperator) GetCloudProvider(ctx context.Context, name string) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudProvider", ctx, name)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCloudProvider indicates an expected call of GetCloudProvider.
func (mr *MockOperatorMockRecorder) GetCloudProvider(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudProvider", reflect.TypeOf((*MockOperator)(nil).GetCloudProvider), ctx, name)
}

// GetCloudProviderEx mocks base method.
func (m *MockOperator) GetCloudProviderEx(ctx context.Context, name, resourceVersion string) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudProviderEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCloudProviderEx indicates an expected call of GetCloudProviderEx.
func (mr *MockOperatorMockRecorder) GetCloudProviderEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudProviderEx", reflect.TypeOf((*MockOperator)(nil).GetCloudProviderEx), ctx, name, resourceVersion)
}

// GetCluster mocks base method.
func (m *MockOperator) GetCluster(ctx context.Context, name string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCluster", ctx, name)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCluster indicates an expected call of GetCluster.
func (mr *MockOperatorMockRecorder) GetCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCluster", reflect.TypeOf((*MockOperator)(nil).GetCluster), ctx, name)
}

// GetClusterEx mocks base method.
func (m *MockOperator) GetClusterEx(ctx context.Context, name, resourceVersion string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterEx indicates an expected call of GetClusterEx.
func (mr *MockOperatorMockRecorder) GetClusterEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterEx", reflect.TypeOf((*MockOperator)(nil).GetClusterEx), ctx, name, resourceVersion)
}

// GetCronBackup mocks base method.
func (m *MockOperator) GetCronBackup(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackup", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackup indicates an expected call of GetCronBackup.
func (mr *MockOperatorMockRecorder) GetCronBackup(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackup", reflect.TypeOf((*MockOperator)(nil).GetCronBackup), ctx, name, resourceVersion)
}

// GetCronBackupEx mocks base method.
func (m *MockOperator) GetCronBackupEx(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackupEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackupEx indicates an expected call of GetCronBackupEx.
func (mr *MockOperatorMockRecorder) GetCronBackupEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackupEx", reflect.TypeOf((*MockOperator)(nil).GetCronBackupEx), ctx, name, resourceVersion)
}

// GetDomain mocks base method.
func (m *MockOperator) GetDomain(ctx context.Context, name string) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomain", ctx, name)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomain indicates an expected call of GetDomain.
func (mr *MockOperatorMockRecorder) GetDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomain", reflect.TypeOf((*MockOperator)(nil).GetDomain), ctx, name)
}

// GetNode mocks base method.
func (m *MockOperator) GetNode(ctx context.Context, name string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNode", ctx, name)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNode indicates an expected call of GetNode.
func (mr *MockOperatorMockRecorder) GetNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNode", reflect.TypeOf((*MockOperator)(nil).GetNode), ctx, name)
}

// GetNodeEx mocks base method.
func (m *MockOperator) GetNodeEx(ctx context.Context, name, resourceVersion string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeEx indicates an expected call of GetNodeEx.
func (mr *MockOperatorMockRecorder) GetNodeEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeEx", reflect.TypeOf((*MockOperator)(nil).GetNodeEx), ctx, name, resourceVersion)
}

// GetRegion mocks base method.
func (m *MockOperator) GetRegion(ctx context.Context, name string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegion", ctx, name)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegion indicates an expected call of GetRegion.
func (mr *MockOperatorMockRecorder) GetRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegion", reflect.TypeOf((*MockOperator)(nil).GetRegion), ctx, name)
}

// GetRegionEx mocks base method.
func (m *MockOperator) GetRegionEx(ctx context.Context, name, resourceVersion string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegionEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegionEx indicates an expected call of GetRegionEx.
func (mr *MockOperatorMockRecorder) GetRegionEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegionEx", reflect.TypeOf((*MockOperator)(nil).GetRegionEx), ctx, name, resourceVersion)
}

// GetRegistry mocks base method.
func (m *MockOperator) GetRegistry(ctx context.Context, name string) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegistry", ctx, name)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegistry indicates an expected call of GetRegistry.
func (mr *MockOperatorMockRecorder) GetRegistry(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegistry", reflect.TypeOf((*MockOperator)(nil).GetRegistry), ctx, name)
}

// GetRegistryEx mocks base method.
func (m *MockOperator) GetRegistryEx(ctx context.Context, name, resourceVersion string) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegistryEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegistryEx indicates an expected call of GetRegistryEx.
func (mr *MockOperatorMockRecorder) GetRegistryEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegistryEx", reflect.TypeOf((*MockOperator)(nil).GetRegistryEx), ctx, name, resourceVersion)
}

// GetTemplate mocks base method.
func (m *MockOperator) GetTemplate(ctx context.Context, name string) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplate", ctx, name)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplate indicates an expected call of GetTemplate.
func (mr *MockOperatorMockRecorder) GetTemplate(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplate", reflect.TypeOf((*MockOperator)(nil).GetTemplate), ctx, name)
}

// GetTemplateEx mocks base method.
func (m *MockOperator) GetTemplateEx(ctx context.Context, name, resourceVersion string) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplateEx indicates an expected call of GetTemplateEx.
func (mr *MockOperatorMockRecorder) GetTemplateEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateEx", reflect.TypeOf((*MockOperator)(nil).GetTemplateEx), ctx, name, resourceVersion)
}

// ListBackupEx mocks base method.
func (m *MockOperator) ListBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupEx indicates an expected call of ListBackupEx.
func (mr *MockOperatorMockRecorder) ListBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupEx", reflect.TypeOf((*MockOperator)(nil).ListBackupEx), ctx, query)
}

// ListBackupPointEx mocks base method.
func (m *MockOperator) ListBackupPointEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPointEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPointEx indicates an expected call of ListBackupPointEx.
func (mr *MockOperatorMockRecorder) ListBackupPointEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPointEx", reflect.TypeOf((*MockOperator)(nil).ListBackupPointEx), ctx, query)
}

// ListBackupPoints mocks base method.
func (m *MockOperator) ListBackupPoints(ctx context.Context, query *query.Query) (*v1.BackupPointList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPoints", ctx, query)
	ret0, _ := ret[0].(*v1.BackupPointList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPoints indicates an expected call of ListBackupPoints.
func (mr *MockOperatorMockRecorder) ListBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPoints", reflect.TypeOf((*MockOperator)(nil).ListBackupPoints), ctx, query)
}

// ListBackups mocks base method.
func (m *MockOperator) ListBackups(ctx context.Context, query *query.Query) (*v1.BackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackups", ctx, query)
	ret0, _ := ret[0].(*v1.BackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackups indicates an expected call of ListBackups.
func (mr *MockOperatorMockRecorder) ListBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackups", reflect.TypeOf((*MockOperator)(nil).ListBackups), ctx, query)
}

// ListCloudProviders mocks base method.
func (m *MockOperator) ListCloudProviders(ctx context.Context, query *query.Query) (*v1.CloudProviderList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCloudProviders", ctx, query)
	ret0, _ := ret[0].(*v1.CloudProviderList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCloudProviders indicates an expected call of ListCloudProviders.
func (mr *MockOperatorMockRecorder) ListCloudProviders(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCloudProviders", reflect.TypeOf((*MockOperator)(nil).ListCloudProviders), ctx, query)
}

// ListCloudProvidersEx mocks base method.
func (m *MockOperator) ListCloudProvidersEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCloudProvidersEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCloudProvidersEx indicates an expected call of ListCloudProvidersEx.
func (mr *MockOperatorMockRecorder) ListCloudProvidersEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCloudProvidersEx", reflect.TypeOf((*MockOperator)(nil).ListCloudProvidersEx), ctx, query)
}

// ListClusterEx mocks base method.
func (m *MockOperator) ListClusterEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusterEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusterEx indicates an expected call of ListClusterEx.
func (mr *MockOperatorMockRecorder) ListClusterEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusterEx", reflect.TypeOf((*MockOperator)(nil).ListClusterEx), ctx, query)
}

// ListClusters mocks base method.
func (m *MockOperator) ListClusters(ctx context.Context, query *query.Query) (*v1.ClusterList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusters", ctx, query)
	ret0, _ := ret[0].(*v1.ClusterList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusters indicates an expected call of ListClusters.
func (mr *MockOperatorMockRecorder) ListClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusters", reflect.TypeOf((*MockOperator)(nil).ListClusters), ctx, query)
}

// ListCronBackupEx mocks base method.
func (m *MockOperator) ListCronBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackupEx indicates an expected call of ListCronBackupEx.
func (mr *MockOperatorMockRecorder) ListCronBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackupEx", reflect.TypeOf((*MockOperator)(nil).ListCronBackupEx), ctx, query)
}

// ListCronBackups mocks base method.
func (m *MockOperator) ListCronBackups(ctx context.Context, query *query.Query) (*v1.CronBackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackups", ctx, query)
	ret0, _ := ret[0].(*v1.CronBackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackups indicates an expected call of ListCronBackups.
func (mr *MockOperatorMockRecorder) ListCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackups", reflect.TypeOf((*MockOperator)(nil).ListCronBackups), ctx, query)
}

// ListDomains mocks base method.
func (m *MockOperator) ListDomains(ctx context.Context, query *query.Query) (*v1.DomainList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomains", ctx, query)
	ret0, _ := ret[0].(*v1.DomainList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomains indicates an expected call of ListDomains.
func (mr *MockOperatorMockRecorder) ListDomains(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomains", reflect.TypeOf((*MockOperator)(nil).ListDomains), ctx, query)
}

// ListDomainsEx mocks base method.
func (m *MockOperator) ListDomainsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomainsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomainsEx indicates an expected call of ListDomainsEx.
func (mr *MockOperatorMockRecorder) ListDomainsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomainsEx", reflect.TypeOf((*MockOperator)(nil).ListDomainsEx), ctx, query)
}

// ListNodes mocks base method.
func (m *MockOperator) ListNodes(ctx context.Context, query *query.Query) (*v1.NodeList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodes", ctx, query)
	ret0, _ := ret[0].(*v1.NodeList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodes indicates an expected call of ListNodes.
func (mr *MockOperatorMockRecorder) ListNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodes", reflect.TypeOf((*MockOperator)(nil).ListNodes), ctx, query)
}

// ListNodesEx mocks base method.
func (m *MockOperator) ListNodesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodesEx indicates an expected call of ListNodesEx.
func (mr *MockOperatorMockRecorder) ListNodesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodesEx", reflect.TypeOf((*MockOperator)(nil).ListNodesEx), ctx, query)
}

// ListRecordsEx mocks base method.
func (m *MockOperator) ListRecordsEx(ctx context.Context, name, subdomain string, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecordsEx", ctx, name, subdomain, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecordsEx indicates an expected call of ListRecordsEx.
func (mr *MockOperatorMockRecorder) ListRecordsEx(ctx, name, subdomain, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecordsEx", reflect.TypeOf((*MockOperator)(nil).ListRecordsEx), ctx, name, subdomain, query)
}

// ListRegionEx mocks base method.
func (m *MockOperator) ListRegionEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegionEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegionEx indicates an expected call of ListRegionEx.
func (mr *MockOperatorMockRecorder) ListRegionEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegionEx", reflect.TypeOf((*MockOperator)(nil).ListRegionEx), ctx, query)
}

// ListRegions mocks base method.
func (m *MockOperator) ListRegions(ctx context.Context, query *query.Query) (*v1.RegionList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegions", ctx, query)
	ret0, _ := ret[0].(*v1.RegionList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegions indicates an expected call of ListRegions.
func (mr *MockOperatorMockRecorder) ListRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegions", reflect.TypeOf((*MockOperator)(nil).ListRegions), ctx, query)
}

// ListRegistries mocks base method.
func (m *MockOperator) ListRegistries(ctx context.Context, query *query.Query) (*v1.RegistryList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegistries", ctx, query)
	ret0, _ := ret[0].(*v1.RegistryList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegistries indicates an expected call of ListRegistries.
func (mr *MockOperatorMockRecorder) ListRegistries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegistries", reflect.TypeOf((*MockOperator)(nil).ListRegistries), ctx, query)
}

// ListRegistriesEx mocks base method.
func (m *MockOperator) ListRegistriesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegistriesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegistriesEx indicates an expected call of ListRegistriesEx.
func (mr *MockOperatorMockRecorder) ListRegistriesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegistriesEx", reflect.TypeOf((*MockOperator)(nil).ListRegistriesEx), ctx, query)
}

// ListTemplates mocks base method.
func (m *MockOperator) ListTemplates(ctx context.Context, query *query.Query) (*v1.TemplateList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplates", ctx, query)
	ret0, _ := ret[0].(*v1.TemplateList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTemplates indicates an expected call of ListTemplates.
func (mr *MockOperatorMockRecorder) ListTemplates(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplates", reflect.TypeOf((*MockOperator)(nil).ListTemplates), ctx, query)
}

// ListTemplatesEx mocks base method.
func (m *MockOperator) ListTemplatesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplatesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTemplatesEx indicates an expected call of ListTemplatesEx.
func (mr *MockOperatorMockRecorder) ListTemplatesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplatesEx", reflect.TypeOf((*MockOperator)(nil).ListTemplatesEx), ctx, query)
}

// UpdateBackup mocks base method.
func (m *MockOperator) UpdateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackup indicates an expected call of UpdateBackup.
func (mr *MockOperatorMockRecorder) UpdateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackup", reflect.TypeOf((*MockOperator)(nil).UpdateBackup), ctx, backup)
}

// UpdateBackupPoint mocks base method.
func (m *MockOperator) UpdateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackupPoint indicates an expected call of UpdateBackupPoint.
func (mr *MockOperatorMockRecorder) UpdateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackupPoint", reflect.TypeOf((*MockOperator)(nil).UpdateBackupPoint), ctx, backupPoint)
}

// UpdateCloudProvider mocks base method.
func (m *MockOperator) UpdateCloudProvider(ctx context.Context, cp *v1.CloudProvider) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCloudProvider", ctx, cp)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCloudProvider indicates an expected call of UpdateCloudProvider.
func (mr *MockOperatorMockRecorder) UpdateCloudProvider(ctx, cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCloudProvider", reflect.TypeOf((*MockOperator)(nil).UpdateCloudProvider), ctx, cp)
}

// UpdateCluster mocks base method.
func (m *MockOperator) UpdateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCluster indicates an expected call of UpdateCluster.
func (mr *MockOperatorMockRecorder) UpdateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCluster", reflect.TypeOf((*MockOperator)(nil).UpdateCluster), ctx, cluster)
}

// UpdateCronBackup mocks base method.
func (m *MockOperator) UpdateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCronBackup indicates an expected call of UpdateCronBackup.
func (mr *MockOperatorMockRecorder) UpdateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCronBackup", reflect.TypeOf((*MockOperator)(nil).UpdateCronBackup), ctx, cronBackup)
}

// UpdateDomain mocks base method.
func (m *MockOperator) UpdateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDomain indicates an expected call of UpdateDomain.
func (mr *MockOperatorMockRecorder) UpdateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDomain", reflect.TypeOf((*MockOperator)(nil).UpdateDomain), ctx, domain)
}

// UpdateNode mocks base method.
func (m *MockOperator) UpdateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateNode indicates an expected call of UpdateNode.
func (mr *MockOperatorMockRecorder) UpdateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNode", reflect.TypeOf((*MockOperator)(nil).UpdateNode), ctx, node)
}

// UpdateRegistry mocks base method.
func (m *MockOperator) UpdateRegistry(ctx context.Context, r *v1.Registry) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRegistry", ctx, r)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRegistry indicates an expected call of UpdateRegistry.
func (mr *MockOperatorMockRecorder) UpdateRegistry(ctx, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRegistry", reflect.TypeOf((*MockOperator)(nil).UpdateRegistry), ctx, r)
}

// UpdateTemplate mocks base method.
func (m *MockOperator) UpdateTemplate(ctx context.Context, template *v1.Template) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTemplate", ctx, template)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateTemplate indicates an expected call of UpdateTemplate.
func (mr *MockOperatorMockRecorder) UpdateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTemplate", reflect.TypeOf((*MockOperator)(nil).UpdateTemplate), ctx, template)
}

// WatchBackupPoints mocks base method.
func (m *MockOperator) WatchBackupPoints(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackupPoints", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackupPoints indicates an expected call of WatchBackupPoints.
func (mr *MockOperatorMockRecorder) WatchBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackupPoints", reflect.TypeOf((*MockOperator)(nil).WatchBackupPoints), ctx, query)
}

// WatchBackups mocks base method.
func (m *MockOperator) WatchBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackups indicates an expected call of WatchBackups.
func (mr *MockOperatorMockRecorder) WatchBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackups", reflect.TypeOf((*MockOperator)(nil).WatchBackups), ctx, query)
}

// WatchCloudProviders mocks base method.
func (m *MockOperator) WatchCloudProviders(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCloudProviders", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCloudProviders indicates an expected call of WatchCloudProviders.
func (mr *MockOperatorMockRecorder) WatchCloudProviders(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCloudProviders", reflect.TypeOf((*MockOperator)(nil).WatchCloudProviders), ctx, query)
}

// WatchClusters mocks base method.
func (m *MockOperator) WatchClusters(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchClusters", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchClusters indicates an expected call of WatchClusters.
func (mr *MockOperatorMockRecorder) WatchClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchClusters", reflect.TypeOf((*MockOperator)(nil).WatchClusters), ctx, query)
}

// WatchCronBackups mocks base method.
func (m *MockOperator) WatchCronBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCronBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCronBackups indicates an expected call of WatchCronBackups.
func (mr *MockOperatorMockRecorder) WatchCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCronBackups", reflect.TypeOf((*MockOperator)(nil).WatchCronBackups), ctx, query)
}

// WatchDomain mocks base method.
func (m *MockOperator) WatchDomain(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchDomain", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchDomain indicates an expected call of WatchDomain.
func (mr *MockOperatorMockRecorder) WatchDomain(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchDomain", reflect.TypeOf((*MockOperator)(nil).WatchDomain), ctx, query)
}

// WatchNodes mocks base method.
func (m *MockOperator) WatchNodes(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchNodes", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchNodes indicates an expected call of WatchNodes.
func (mr *MockOperatorMockRecorder) WatchNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchNodes", reflect.TypeOf((*MockOperator)(nil).WatchNodes), ctx, query)
}

// WatchRegions mocks base method.
func (m *MockOperator) WatchRegions(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRegions", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRegions indicates an expected call of WatchRegions.
func (mr *MockOperatorMockRecorder) WatchRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRegions", reflect.TypeOf((*MockOperator)(nil).WatchRegions), ctx, query)
}

// WatchRegistries mocks base method.
func (m *MockOperator) WatchRegistries(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRegistries", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRegistries indicates an expected call of WatchRegistries.
func (mr *MockOperatorMockRecorder) WatchRegistries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRegistries", reflect.TypeOf((*MockOperator)(nil).WatchRegistries), ctx, query)
}

// WatchTemplates mocks base method.
func (m *MockOperator) WatchTemplates(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchTemplates", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchTemplates indicates an expected call of WatchTemplates.
func (mr *MockOperatorMockRecorder) WatchTemplates(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchTemplates", reflect.TypeOf((*MockOperator)(nil).WatchTemplates), ctx, query)
}

// MockClusterReader is a mock of ClusterReader interface.
type MockClusterReader struct {
	ctrl     *gomock.Controller
	recorder *MockClusterReaderMockRecorder
}

// MockClusterReaderMockRecorder is the mock recorder for MockClusterReader.
type MockClusterReaderMockRecorder struct {
	mock *MockClusterReader
}

// NewMockClusterReader creates a new mock instance.
func NewMockClusterReader(ctrl *gomock.Controller) *MockClusterReader {
	mock := &MockClusterReader{ctrl: ctrl}
	mock.recorder = &MockClusterReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterReader) EXPECT() *MockClusterReaderMockRecorder {
	return m.recorder
}

// GetCluster mocks base method.
func (m *MockClusterReader) GetCluster(ctx context.Context, name string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCluster", ctx, name)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCluster indicates an expected call of GetCluster.
func (mr *MockClusterReaderMockRecorder) GetCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCluster", reflect.TypeOf((*MockClusterReader)(nil).GetCluster), ctx, name)
}

// GetClusterEx mocks base method.
func (m *MockClusterReader) GetClusterEx(ctx context.Context, name, resourceVersion string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterEx indicates an expected call of GetClusterEx.
func (mr *MockClusterReaderMockRecorder) GetClusterEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterEx", reflect.TypeOf((*MockClusterReader)(nil).GetClusterEx), ctx, name, resourceVersion)
}

// ListClusterEx mocks base method.
func (m *MockClusterReader) ListClusterEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusterEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusterEx indicates an expected call of ListClusterEx.
func (mr *MockClusterReaderMockRecorder) ListClusterEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusterEx", reflect.TypeOf((*MockClusterReader)(nil).ListClusterEx), ctx, query)
}

// ListClusters mocks base method.
func (m *MockClusterReader) ListClusters(ctx context.Context, query *query.Query) (*v1.ClusterList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusters", ctx, query)
	ret0, _ := ret[0].(*v1.ClusterList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusters indicates an expected call of ListClusters.
func (mr *MockClusterReaderMockRecorder) ListClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusters", reflect.TypeOf((*MockClusterReader)(nil).ListClusters), ctx, query)
}

// WatchClusters mocks base method.
func (m *MockClusterReader) WatchClusters(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchClusters", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchClusters indicates an expected call of WatchClusters.
func (mr *MockClusterReaderMockRecorder) WatchClusters(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchClusters", reflect.TypeOf((*MockClusterReader)(nil).WatchClusters), ctx, query)
}

// MockClusterReaderEx is a mock of ClusterReaderEx interface.
type MockClusterReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockClusterReaderExMockRecorder
}

// MockClusterReaderExMockRecorder is the mock recorder for MockClusterReaderEx.
type MockClusterReaderExMockRecorder struct {
	mock *MockClusterReaderEx
}

// NewMockClusterReaderEx creates a new mock instance.
func NewMockClusterReaderEx(ctrl *gomock.Controller) *MockClusterReaderEx {
	mock := &MockClusterReaderEx{ctrl: ctrl}
	mock.recorder = &MockClusterReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterReaderEx) EXPECT() *MockClusterReaderExMockRecorder {
	return m.recorder
}

// GetClusterEx mocks base method.
func (m *MockClusterReaderEx) GetClusterEx(ctx context.Context, name, resourceVersion string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterEx indicates an expected call of GetClusterEx.
func (mr *MockClusterReaderExMockRecorder) GetClusterEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterEx", reflect.TypeOf((*MockClusterReaderEx)(nil).GetClusterEx), ctx, name, resourceVersion)
}

// ListClusterEx mocks base method.
func (m *MockClusterReaderEx) ListClusterEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClusterEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClusterEx indicates an expected call of ListClusterEx.
func (mr *MockClusterReaderExMockRecorder) ListClusterEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClusterEx", reflect.TypeOf((*MockClusterReaderEx)(nil).ListClusterEx), ctx, query)
}

// MockClusterWriter is a mock of ClusterWriter interface.
type MockClusterWriter struct {
	ctrl     *gomock.Controller
	recorder *MockClusterWriterMockRecorder
}

// MockClusterWriterMockRecorder is the mock recorder for MockClusterWriter.
type MockClusterWriterMockRecorder struct {
	mock *MockClusterWriter
}

// NewMockClusterWriter creates a new mock instance.
func NewMockClusterWriter(ctrl *gomock.Controller) *MockClusterWriter {
	mock := &MockClusterWriter{ctrl: ctrl}
	mock.recorder = &MockClusterWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterWriter) EXPECT() *MockClusterWriterMockRecorder {
	return m.recorder
}

// CreateCluster mocks base method.
func (m *MockClusterWriter) CreateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCluster indicates an expected call of CreateCluster.
func (mr *MockClusterWriterMockRecorder) CreateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCluster", reflect.TypeOf((*MockClusterWriter)(nil).CreateCluster), ctx, cluster)
}

// DeleteCluster mocks base method.
func (m *MockClusterWriter) DeleteCluster(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCluster", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCluster indicates an expected call of DeleteCluster.
func (mr *MockClusterWriterMockRecorder) DeleteCluster(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCluster", reflect.TypeOf((*MockClusterWriter)(nil).DeleteCluster), ctx, name)
}

// UpdateCluster mocks base method.
func (m *MockClusterWriter) UpdateCluster(ctx context.Context, cluster *v1.Cluster) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCluster", ctx, cluster)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCluster indicates an expected call of UpdateCluster.
func (mr *MockClusterWriterMockRecorder) UpdateCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCluster", reflect.TypeOf((*MockClusterWriter)(nil).UpdateCluster), ctx, cluster)
}

// MockRegionReader is a mock of RegionReader interface.
type MockRegionReader struct {
	ctrl     *gomock.Controller
	recorder *MockRegionReaderMockRecorder
}

// MockRegionReaderMockRecorder is the mock recorder for MockRegionReader.
type MockRegionReaderMockRecorder struct {
	mock *MockRegionReader
}

// NewMockRegionReader creates a new mock instance.
func NewMockRegionReader(ctrl *gomock.Controller) *MockRegionReader {
	mock := &MockRegionReader{ctrl: ctrl}
	mock.recorder = &MockRegionReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegionReader) EXPECT() *MockRegionReaderMockRecorder {
	return m.recorder
}

// GetRegion mocks base method.
func (m *MockRegionReader) GetRegion(ctx context.Context, name string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegion", ctx, name)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegion indicates an expected call of GetRegion.
func (mr *MockRegionReaderMockRecorder) GetRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegion", reflect.TypeOf((*MockRegionReader)(nil).GetRegion), ctx, name)
}

// GetRegionEx mocks base method.
func (m *MockRegionReader) GetRegionEx(ctx context.Context, name, resourceVersion string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegionEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegionEx indicates an expected call of GetRegionEx.
func (mr *MockRegionReaderMockRecorder) GetRegionEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegionEx", reflect.TypeOf((*MockRegionReader)(nil).GetRegionEx), ctx, name, resourceVersion)
}

// ListRegionEx mocks base method.
func (m *MockRegionReader) ListRegionEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegionEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegionEx indicates an expected call of ListRegionEx.
func (mr *MockRegionReaderMockRecorder) ListRegionEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegionEx", reflect.TypeOf((*MockRegionReader)(nil).ListRegionEx), ctx, query)
}

// ListRegions mocks base method.
func (m *MockRegionReader) ListRegions(ctx context.Context, query *query.Query) (*v1.RegionList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegions", ctx, query)
	ret0, _ := ret[0].(*v1.RegionList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegions indicates an expected call of ListRegions.
func (mr *MockRegionReaderMockRecorder) ListRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegions", reflect.TypeOf((*MockRegionReader)(nil).ListRegions), ctx, query)
}

// WatchRegions mocks base method.
func (m *MockRegionReader) WatchRegions(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRegions", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRegions indicates an expected call of WatchRegions.
func (mr *MockRegionReaderMockRecorder) WatchRegions(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRegions", reflect.TypeOf((*MockRegionReader)(nil).WatchRegions), ctx, query)
}

// MockRegionReaderEx is a mock of RegionReaderEx interface.
type MockRegionReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockRegionReaderExMockRecorder
}

// MockRegionReaderExMockRecorder is the mock recorder for MockRegionReaderEx.
type MockRegionReaderExMockRecorder struct {
	mock *MockRegionReaderEx
}

// NewMockRegionReaderEx creates a new mock instance.
func NewMockRegionReaderEx(ctrl *gomock.Controller) *MockRegionReaderEx {
	mock := &MockRegionReaderEx{ctrl: ctrl}
	mock.recorder = &MockRegionReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegionReaderEx) EXPECT() *MockRegionReaderExMockRecorder {
	return m.recorder
}

// GetRegionEx mocks base method.
func (m *MockRegionReaderEx) GetRegionEx(ctx context.Context, name, resourceVersion string) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegionEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegionEx indicates an expected call of GetRegionEx.
func (mr *MockRegionReaderExMockRecorder) GetRegionEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegionEx", reflect.TypeOf((*MockRegionReaderEx)(nil).GetRegionEx), ctx, name, resourceVersion)
}

// ListRegionEx mocks base method.
func (m *MockRegionReaderEx) ListRegionEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegionEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegionEx indicates an expected call of ListRegionEx.
func (mr *MockRegionReaderExMockRecorder) ListRegionEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegionEx", reflect.TypeOf((*MockRegionReaderEx)(nil).ListRegionEx), ctx, query)
}

// MockRegionWriter is a mock of RegionWriter interface.
type MockRegionWriter struct {
	ctrl     *gomock.Controller
	recorder *MockRegionWriterMockRecorder
}

// MockRegionWriterMockRecorder is the mock recorder for MockRegionWriter.
type MockRegionWriterMockRecorder struct {
	mock *MockRegionWriter
}

// NewMockRegionWriter creates a new mock instance.
func NewMockRegionWriter(ctrl *gomock.Controller) *MockRegionWriter {
	mock := &MockRegionWriter{ctrl: ctrl}
	mock.recorder = &MockRegionWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegionWriter) EXPECT() *MockRegionWriterMockRecorder {
	return m.recorder
}

// CreateRegion mocks base method.
func (m *MockRegionWriter) CreateRegion(ctx context.Context, region *v1.Region) (*v1.Region, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRegion", ctx, region)
	ret0, _ := ret[0].(*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRegion indicates an expected call of CreateRegion.
func (mr *MockRegionWriterMockRecorder) CreateRegion(ctx, region interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRegion", reflect.TypeOf((*MockRegionWriter)(nil).CreateRegion), ctx, region)
}

// DeleteRegion mocks base method.
func (m *MockRegionWriter) DeleteRegion(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRegion", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRegion indicates an expected call of DeleteRegion.
func (mr *MockRegionWriterMockRecorder) DeleteRegion(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRegion", reflect.TypeOf((*MockRegionWriter)(nil).DeleteRegion), ctx, name)
}

// MockNodeReader is a mock of NodeReader interface.
type MockNodeReader struct {
	ctrl     *gomock.Controller
	recorder *MockNodeReaderMockRecorder
}

// MockNodeReaderMockRecorder is the mock recorder for MockNodeReader.
type MockNodeReaderMockRecorder struct {
	mock *MockNodeReader
}

// NewMockNodeReader creates a new mock instance.
func NewMockNodeReader(ctrl *gomock.Controller) *MockNodeReader {
	mock := &MockNodeReader{ctrl: ctrl}
	mock.recorder = &MockNodeReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNodeReader) EXPECT() *MockNodeReaderMockRecorder {
	return m.recorder
}

// GetNode mocks base method.
func (m *MockNodeReader) GetNode(ctx context.Context, name string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNode", ctx, name)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNode indicates an expected call of GetNode.
func (mr *MockNodeReaderMockRecorder) GetNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNode", reflect.TypeOf((*MockNodeReader)(nil).GetNode), ctx, name)
}

// GetNodeEx mocks base method.
func (m *MockNodeReader) GetNodeEx(ctx context.Context, name, resourceVersion string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeEx indicates an expected call of GetNodeEx.
func (mr *MockNodeReaderMockRecorder) GetNodeEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeEx", reflect.TypeOf((*MockNodeReader)(nil).GetNodeEx), ctx, name, resourceVersion)
}

// ListNodes mocks base method.
func (m *MockNodeReader) ListNodes(ctx context.Context, query *query.Query) (*v1.NodeList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodes", ctx, query)
	ret0, _ := ret[0].(*v1.NodeList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodes indicates an expected call of ListNodes.
func (mr *MockNodeReaderMockRecorder) ListNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodes", reflect.TypeOf((*MockNodeReader)(nil).ListNodes), ctx, query)
}

// ListNodesEx mocks base method.
func (m *MockNodeReader) ListNodesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodesEx indicates an expected call of ListNodesEx.
func (mr *MockNodeReaderMockRecorder) ListNodesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodesEx", reflect.TypeOf((*MockNodeReader)(nil).ListNodesEx), ctx, query)
}

// WatchNodes mocks base method.
func (m *MockNodeReader) WatchNodes(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchNodes", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchNodes indicates an expected call of WatchNodes.
func (mr *MockNodeReaderMockRecorder) WatchNodes(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchNodes", reflect.TypeOf((*MockNodeReader)(nil).WatchNodes), ctx, query)
}

// MockNodeReaderEx is a mock of NodeReaderEx interface.
type MockNodeReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockNodeReaderExMockRecorder
}

// MockNodeReaderExMockRecorder is the mock recorder for MockNodeReaderEx.
type MockNodeReaderExMockRecorder struct {
	mock *MockNodeReaderEx
}

// NewMockNodeReaderEx creates a new mock instance.
func NewMockNodeReaderEx(ctrl *gomock.Controller) *MockNodeReaderEx {
	mock := &MockNodeReaderEx{ctrl: ctrl}
	mock.recorder = &MockNodeReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNodeReaderEx) EXPECT() *MockNodeReaderExMockRecorder {
	return m.recorder
}

// GetNodeEx mocks base method.
func (m *MockNodeReaderEx) GetNodeEx(ctx context.Context, name, resourceVersion string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeEx indicates an expected call of GetNodeEx.
func (mr *MockNodeReaderExMockRecorder) GetNodeEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeEx", reflect.TypeOf((*MockNodeReaderEx)(nil).GetNodeEx), ctx, name, resourceVersion)
}

// ListNodesEx mocks base method.
func (m *MockNodeReaderEx) ListNodesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodesEx indicates an expected call of ListNodesEx.
func (mr *MockNodeReaderExMockRecorder) ListNodesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodesEx", reflect.TypeOf((*MockNodeReaderEx)(nil).ListNodesEx), ctx, query)
}

// MockNodeWriter is a mock of NodeWriter interface.
type MockNodeWriter struct {
	ctrl     *gomock.Controller
	recorder *MockNodeWriterMockRecorder
}

// MockNodeWriterMockRecorder is the mock recorder for MockNodeWriter.
type MockNodeWriterMockRecorder struct {
	mock *MockNodeWriter
}

// NewMockNodeWriter creates a new mock instance.
func NewMockNodeWriter(ctrl *gomock.Controller) *MockNodeWriter {
	mock := &MockNodeWriter{ctrl: ctrl}
	mock.recorder = &MockNodeWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNodeWriter) EXPECT() *MockNodeWriterMockRecorder {
	return m.recorder
}

// CreateNode mocks base method.
func (m *MockNodeWriter) CreateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNode indicates an expected call of CreateNode.
func (mr *MockNodeWriterMockRecorder) CreateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNode", reflect.TypeOf((*MockNodeWriter)(nil).CreateNode), ctx, node)
}

// DeleteNode mocks base method.
func (m *MockNodeWriter) DeleteNode(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNode", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNode indicates an expected call of DeleteNode.
func (mr *MockNodeWriterMockRecorder) DeleteNode(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNode", reflect.TypeOf((*MockNodeWriter)(nil).DeleteNode), ctx, name)
}

// UpdateNode mocks base method.
func (m *MockNodeWriter) UpdateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNode", ctx, node)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateNode indicates an expected call of UpdateNode.
func (mr *MockNodeWriterMockRecorder) UpdateNode(ctx, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNode", reflect.TypeOf((*MockNodeWriter)(nil).UpdateNode), ctx, node)
}

// MockBackupReader is a mock of BackupReader interface.
type MockBackupReader struct {
	ctrl     *gomock.Controller
	recorder *MockBackupReaderMockRecorder
}

// MockBackupReaderMockRecorder is the mock recorder for MockBackupReader.
type MockBackupReaderMockRecorder struct {
	mock *MockBackupReader
}

// NewMockBackupReader creates a new mock instance.
func NewMockBackupReader(ctrl *gomock.Controller) *MockBackupReader {
	mock := &MockBackupReader{ctrl: ctrl}
	mock.recorder = &MockBackupReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupReader) EXPECT() *MockBackupReaderMockRecorder {
	return m.recorder
}

// GetBackup mocks base method.
func (m *MockBackupReader) GetBackup(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackup", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackup indicates an expected call of GetBackup.
func (mr *MockBackupReaderMockRecorder) GetBackup(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackup", reflect.TypeOf((*MockBackupReader)(nil).GetBackup), ctx, cluster, name)
}

// GetBackupEx mocks base method.
func (m *MockBackupReader) GetBackupEx(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupEx indicates an expected call of GetBackupEx.
func (mr *MockBackupReaderMockRecorder) GetBackupEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupEx", reflect.TypeOf((*MockBackupReader)(nil).GetBackupEx), ctx, cluster, name)
}

// ListBackupEx mocks base method.
func (m *MockBackupReader) ListBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupEx indicates an expected call of ListBackupEx.
func (mr *MockBackupReaderMockRecorder) ListBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupEx", reflect.TypeOf((*MockBackupReader)(nil).ListBackupEx), ctx, query)
}

// ListBackups mocks base method.
func (m *MockBackupReader) ListBackups(ctx context.Context, query *query.Query) (*v1.BackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackups", ctx, query)
	ret0, _ := ret[0].(*v1.BackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackups indicates an expected call of ListBackups.
func (mr *MockBackupReaderMockRecorder) ListBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackups", reflect.TypeOf((*MockBackupReader)(nil).ListBackups), ctx, query)
}

// WatchBackups mocks base method.
func (m *MockBackupReader) WatchBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackups indicates an expected call of WatchBackups.
func (mr *MockBackupReaderMockRecorder) WatchBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackups", reflect.TypeOf((*MockBackupReader)(nil).WatchBackups), ctx, query)
}

// MockBackupReaderEx is a mock of BackupReaderEx interface.
type MockBackupReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockBackupReaderExMockRecorder
}

// MockBackupReaderExMockRecorder is the mock recorder for MockBackupReaderEx.
type MockBackupReaderExMockRecorder struct {
	mock *MockBackupReaderEx
}

// NewMockBackupReaderEx creates a new mock instance.
func NewMockBackupReaderEx(ctrl *gomock.Controller) *MockBackupReaderEx {
	mock := &MockBackupReaderEx{ctrl: ctrl}
	mock.recorder = &MockBackupReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupReaderEx) EXPECT() *MockBackupReaderExMockRecorder {
	return m.recorder
}

// GetBackupEx mocks base method.
func (m *MockBackupReaderEx) GetBackupEx(ctx context.Context, cluster, name string) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupEx indicates an expected call of GetBackupEx.
func (mr *MockBackupReaderExMockRecorder) GetBackupEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupEx", reflect.TypeOf((*MockBackupReaderEx)(nil).GetBackupEx), ctx, cluster, name)
}

// ListBackupEx mocks base method.
func (m *MockBackupReaderEx) ListBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupEx indicates an expected call of ListBackupEx.
func (mr *MockBackupReaderExMockRecorder) ListBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupEx", reflect.TypeOf((*MockBackupReaderEx)(nil).ListBackupEx), ctx, query)
}

// MockBackupWriter is a mock of BackupWriter interface.
type MockBackupWriter struct {
	ctrl     *gomock.Controller
	recorder *MockBackupWriterMockRecorder
}

// MockBackupWriterMockRecorder is the mock recorder for MockBackupWriter.
type MockBackupWriterMockRecorder struct {
	mock *MockBackupWriter
}

// NewMockBackupWriter creates a new mock instance.
func NewMockBackupWriter(ctrl *gomock.Controller) *MockBackupWriter {
	mock := &MockBackupWriter{ctrl: ctrl}
	mock.recorder = &MockBackupWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupWriter) EXPECT() *MockBackupWriterMockRecorder {
	return m.recorder
}

// CreateBackup mocks base method.
func (m *MockBackupWriter) CreateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackup indicates an expected call of CreateBackup.
func (mr *MockBackupWriterMockRecorder) CreateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackup", reflect.TypeOf((*MockBackupWriter)(nil).CreateBackup), ctx, backup)
}

// DeleteBackup mocks base method.
func (m *MockBackupWriter) DeleteBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackup indicates an expected call of DeleteBackup.
func (mr *MockBackupWriterMockRecorder) DeleteBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackup", reflect.TypeOf((*MockBackupWriter)(nil).DeleteBackup), ctx, name)
}

// UpdateBackup mocks base method.
func (m *MockBackupWriter) UpdateBackup(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackup", ctx, backup)
	ret0, _ := ret[0].(*v1.Backup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackup indicates an expected call of UpdateBackup.
func (mr *MockBackupWriterMockRecorder) UpdateBackup(ctx, backup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackup", reflect.TypeOf((*MockBackupWriter)(nil).UpdateBackup), ctx, backup)
}

// MockRecoveryReader is a mock of RecoveryReader interface.
type MockRecoveryReader struct {
	ctrl     *gomock.Controller
	recorder *MockRecoveryReaderMockRecorder
}

// MockRecoveryReaderMockRecorder is the mock recorder for MockRecoveryReader.
type MockRecoveryReaderMockRecorder struct {
	mock *MockRecoveryReader
}

// NewMockRecoveryReader creates a new mock instance.
func NewMockRecoveryReader(ctrl *gomock.Controller) *MockRecoveryReader {
	mock := &MockRecoveryReader{ctrl: ctrl}
	mock.recorder = &MockRecoveryReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRecoveryReader) EXPECT() *MockRecoveryReaderMockRecorder {
	return m.recorder
}

// GetRecovery mocks base method.
func (m *MockRecoveryReader) GetRecovery(ctx context.Context, cluster, name string) (*v1.Recovery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRecovery", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Recovery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRecovery indicates an expected call of GetRecovery.
func (mr *MockRecoveryReaderMockRecorder) GetRecovery(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRecovery", reflect.TypeOf((*MockRecoveryReader)(nil).GetRecovery), ctx, cluster, name)
}

// GetRecoveryEx mocks base method.
func (m *MockRecoveryReader) GetRecoveryEx(ctx context.Context, cluster, name string) (*v1.Recovery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRecoveryEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Recovery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRecoveryEx indicates an expected call of GetRecoveryEx.
func (mr *MockRecoveryReaderMockRecorder) GetRecoveryEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRecoveryEx", reflect.TypeOf((*MockRecoveryReader)(nil).GetRecoveryEx), ctx, cluster, name)
}

// ListRecoveries mocks base method.
func (m *MockRecoveryReader) ListRecoveries(ctx context.Context, query *query.Query) (*v1.RecoveryList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecoveries", ctx, query)
	ret0, _ := ret[0].(*v1.RecoveryList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecoveries indicates an expected call of ListRecoveries.
func (mr *MockRecoveryReaderMockRecorder) ListRecoveries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecoveries", reflect.TypeOf((*MockRecoveryReader)(nil).ListRecoveries), ctx, query)
}

// ListRecoveryEx mocks base method.
func (m *MockRecoveryReader) ListRecoveryEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecoveryEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecoveryEx indicates an expected call of ListRecoveryEx.
func (mr *MockRecoveryReaderMockRecorder) ListRecoveryEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecoveryEx", reflect.TypeOf((*MockRecoveryReader)(nil).ListRecoveryEx), ctx, query)
}

// WatchRecoveries mocks base method.
func (m *MockRecoveryReader) WatchRecoveries(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRecoveries", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRecoveries indicates an expected call of WatchRecoveries.
func (mr *MockRecoveryReaderMockRecorder) WatchRecoveries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRecoveries", reflect.TypeOf((*MockRecoveryReader)(nil).WatchRecoveries), ctx, query)
}

// MockRecoveryReaderEx is a mock of RecoveryReaderEx interface.
type MockRecoveryReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockRecoveryReaderExMockRecorder
}

// MockRecoveryReaderExMockRecorder is the mock recorder for MockRecoveryReaderEx.
type MockRecoveryReaderExMockRecorder struct {
	mock *MockRecoveryReaderEx
}

// NewMockRecoveryReaderEx creates a new mock instance.
func NewMockRecoveryReaderEx(ctrl *gomock.Controller) *MockRecoveryReaderEx {
	mock := &MockRecoveryReaderEx{ctrl: ctrl}
	mock.recorder = &MockRecoveryReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRecoveryReaderEx) EXPECT() *MockRecoveryReaderExMockRecorder {
	return m.recorder
}

// GetRecoveryEx mocks base method.
func (m *MockRecoveryReaderEx) GetRecoveryEx(ctx context.Context, cluster, name string) (*v1.Recovery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRecoveryEx", ctx, cluster, name)
	ret0, _ := ret[0].(*v1.Recovery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRecoveryEx indicates an expected call of GetRecoveryEx.
func (mr *MockRecoveryReaderExMockRecorder) GetRecoveryEx(ctx, cluster, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRecoveryEx", reflect.TypeOf((*MockRecoveryReaderEx)(nil).GetRecoveryEx), ctx, cluster, name)
}

// ListRecoveryEx mocks base method.
func (m *MockRecoveryReaderEx) ListRecoveryEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecoveryEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecoveryEx indicates an expected call of ListRecoveryEx.
func (mr *MockRecoveryReaderExMockRecorder) ListRecoveryEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecoveryEx", reflect.TypeOf((*MockRecoveryReaderEx)(nil).ListRecoveryEx), ctx, query)
}

// MockRecoveryWriter is a mock of RecoveryWriter interface.
type MockRecoveryWriter struct {
	ctrl     *gomock.Controller
	recorder *MockRecoveryWriterMockRecorder
}

// MockRecoveryWriterMockRecorder is the mock recorder for MockRecoveryWriter.
type MockRecoveryWriterMockRecorder struct {
	mock *MockRecoveryWriter
}

// NewMockRecoveryWriter creates a new mock instance.
func NewMockRecoveryWriter(ctrl *gomock.Controller) *MockRecoveryWriter {
	mock := &MockRecoveryWriter{ctrl: ctrl}
	mock.recorder = &MockRecoveryWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRecoveryWriter) EXPECT() *MockRecoveryWriterMockRecorder {
	return m.recorder
}

// CreateRecovery mocks base method.
func (m *MockRecoveryWriter) CreateRecovery(ctx context.Context, recovery *v1.Recovery) (*v1.Recovery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRecovery", ctx, recovery)
	ret0, _ := ret[0].(*v1.Recovery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRecovery indicates an expected call of CreateRecovery.
func (mr *MockRecoveryWriterMockRecorder) CreateRecovery(ctx, recovery interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRecovery", reflect.TypeOf((*MockRecoveryWriter)(nil).CreateRecovery), ctx, recovery)
}

// DeleteRecovery mocks base method.
func (m *MockRecoveryWriter) DeleteRecovery(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRecovery", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRecovery indicates an expected call of DeleteRecovery.
func (mr *MockRecoveryWriterMockRecorder) DeleteRecovery(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRecovery", reflect.TypeOf((*MockRecoveryWriter)(nil).DeleteRecovery), ctx, name)
}

// UpdateRecovery mocks base method.
func (m *MockRecoveryWriter) UpdateRecovery(ctx context.Context, recovery *v1.Recovery) (*v1.Recovery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRecovery", ctx, recovery)
	ret0, _ := ret[0].(*v1.Recovery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRecovery indicates an expected call of UpdateRecovery.
func (mr *MockRecoveryWriterMockRecorder) UpdateRecovery(ctx, recovery interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRecovery", reflect.TypeOf((*MockRecoveryWriter)(nil).UpdateRecovery), ctx, recovery)
}

// MockBackupPointReaderEx is a mock of BackupPointReaderEx interface.
type MockBackupPointReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockBackupPointReaderExMockRecorder
}

// MockBackupPointReaderExMockRecorder is the mock recorder for MockBackupPointReaderEx.
type MockBackupPointReaderExMockRecorder struct {
	mock *MockBackupPointReaderEx
}

// NewMockBackupPointReaderEx creates a new mock instance.
func NewMockBackupPointReaderEx(ctrl *gomock.Controller) *MockBackupPointReaderEx {
	mock := &MockBackupPointReaderEx{ctrl: ctrl}
	mock.recorder = &MockBackupPointReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupPointReaderEx) EXPECT() *MockBackupPointReaderExMockRecorder {
	return m.recorder
}

// GetBackupPointEx mocks base method.
func (m *MockBackupPointReaderEx) GetBackupPointEx(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPointEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPointEx indicates an expected call of GetBackupPointEx.
func (mr *MockBackupPointReaderExMockRecorder) GetBackupPointEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPointEx", reflect.TypeOf((*MockBackupPointReaderEx)(nil).GetBackupPointEx), ctx, name, resourceVersion)
}

// ListBackupPointEx mocks base method.
func (m *MockBackupPointReaderEx) ListBackupPointEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPointEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPointEx indicates an expected call of ListBackupPointEx.
func (mr *MockBackupPointReaderExMockRecorder) ListBackupPointEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPointEx", reflect.TypeOf((*MockBackupPointReaderEx)(nil).ListBackupPointEx), ctx, query)
}

// MockBackupPointReader is a mock of BackupPointReader interface.
type MockBackupPointReader struct {
	ctrl     *gomock.Controller
	recorder *MockBackupPointReaderMockRecorder
}

// MockBackupPointReaderMockRecorder is the mock recorder for MockBackupPointReader.
type MockBackupPointReaderMockRecorder struct {
	mock *MockBackupPointReader
}

// NewMockBackupPointReader creates a new mock instance.
func NewMockBackupPointReader(ctrl *gomock.Controller) *MockBackupPointReader {
	mock := &MockBackupPointReader{ctrl: ctrl}
	mock.recorder = &MockBackupPointReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupPointReader) EXPECT() *MockBackupPointReaderMockRecorder {
	return m.recorder
}

// GetBackupPoint mocks base method.
func (m *MockBackupPointReader) GetBackupPoint(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPoint", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPoint indicates an expected call of GetBackupPoint.
func (mr *MockBackupPointReaderMockRecorder) GetBackupPoint(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPoint", reflect.TypeOf((*MockBackupPointReader)(nil).GetBackupPoint), ctx, name, resourceVersion)
}

// GetBackupPointEx mocks base method.
func (m *MockBackupPointReader) GetBackupPointEx(ctx context.Context, name, resourceVersion string) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBackupPointEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBackupPointEx indicates an expected call of GetBackupPointEx.
func (mr *MockBackupPointReaderMockRecorder) GetBackupPointEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBackupPointEx", reflect.TypeOf((*MockBackupPointReader)(nil).GetBackupPointEx), ctx, name, resourceVersion)
}

// ListBackupPointEx mocks base method.
func (m *MockBackupPointReader) ListBackupPointEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPointEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPointEx indicates an expected call of ListBackupPointEx.
func (mr *MockBackupPointReaderMockRecorder) ListBackupPointEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPointEx", reflect.TypeOf((*MockBackupPointReader)(nil).ListBackupPointEx), ctx, query)
}

// ListBackupPoints mocks base method.
func (m *MockBackupPointReader) ListBackupPoints(ctx context.Context, query *query.Query) (*v1.BackupPointList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBackupPoints", ctx, query)
	ret0, _ := ret[0].(*v1.BackupPointList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBackupPoints indicates an expected call of ListBackupPoints.
func (mr *MockBackupPointReaderMockRecorder) ListBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBackupPoints", reflect.TypeOf((*MockBackupPointReader)(nil).ListBackupPoints), ctx, query)
}

// WatchBackupPoints mocks base method.
func (m *MockBackupPointReader) WatchBackupPoints(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchBackupPoints", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchBackupPoints indicates an expected call of WatchBackupPoints.
func (mr *MockBackupPointReaderMockRecorder) WatchBackupPoints(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchBackupPoints", reflect.TypeOf((*MockBackupPointReader)(nil).WatchBackupPoints), ctx, query)
}

// MockBackupPointWriter is a mock of BackupPointWriter interface.
type MockBackupPointWriter struct {
	ctrl     *gomock.Controller
	recorder *MockBackupPointWriterMockRecorder
}

// MockBackupPointWriterMockRecorder is the mock recorder for MockBackupPointWriter.
type MockBackupPointWriterMockRecorder struct {
	mock *MockBackupPointWriter
}

// NewMockBackupPointWriter creates a new mock instance.
func NewMockBackupPointWriter(ctrl *gomock.Controller) *MockBackupPointWriter {
	mock := &MockBackupPointWriter{ctrl: ctrl}
	mock.recorder = &MockBackupPointWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupPointWriter) EXPECT() *MockBackupPointWriterMockRecorder {
	return m.recorder
}

// CreateBackupPoint mocks base method.
func (m *MockBackupPointWriter) CreateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackupPoint indicates an expected call of CreateBackupPoint.
func (mr *MockBackupPointWriterMockRecorder) CreateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackupPoint", reflect.TypeOf((*MockBackupPointWriter)(nil).CreateBackupPoint), ctx, backupPoint)
}

// DeleteBackupPoint mocks base method.
func (m *MockBackupPointWriter) DeleteBackupPoint(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBackupPoint", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBackupPoint indicates an expected call of DeleteBackupPoint.
func (mr *MockBackupPointWriterMockRecorder) DeleteBackupPoint(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBackupPoint", reflect.TypeOf((*MockBackupPointWriter)(nil).DeleteBackupPoint), ctx, name)
}

// UpdateBackupPoint mocks base method.
func (m *MockBackupPointWriter) UpdateBackupPoint(ctx context.Context, backupPoint *v1.BackupPoint) (*v1.BackupPoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBackupPoint", ctx, backupPoint)
	ret0, _ := ret[0].(*v1.BackupPoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBackupPoint indicates an expected call of UpdateBackupPoint.
func (mr *MockBackupPointWriterMockRecorder) UpdateBackupPoint(ctx, backupPoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBackupPoint", reflect.TypeOf((*MockBackupPointWriter)(nil).UpdateBackupPoint), ctx, backupPoint)
}

// MockCronBackupReaderEx is a mock of CronBackupReaderEx interface.
type MockCronBackupReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockCronBackupReaderExMockRecorder
}

// MockCronBackupReaderExMockRecorder is the mock recorder for MockCronBackupReaderEx.
type MockCronBackupReaderExMockRecorder struct {
	mock *MockCronBackupReaderEx
}

// NewMockCronBackupReaderEx creates a new mock instance.
func NewMockCronBackupReaderEx(ctrl *gomock.Controller) *MockCronBackupReaderEx {
	mock := &MockCronBackupReaderEx{ctrl: ctrl}
	mock.recorder = &MockCronBackupReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCronBackupReaderEx) EXPECT() *MockCronBackupReaderExMockRecorder {
	return m.recorder
}

// GetCronBackupEx mocks base method.
func (m *MockCronBackupReaderEx) GetCronBackupEx(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackupEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackupEx indicates an expected call of GetCronBackupEx.
func (mr *MockCronBackupReaderExMockRecorder) GetCronBackupEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackupEx", reflect.TypeOf((*MockCronBackupReaderEx)(nil).GetCronBackupEx), ctx, name, resourceVersion)
}

// ListCronBackupEx mocks base method.
func (m *MockCronBackupReaderEx) ListCronBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackupEx indicates an expected call of ListCronBackupEx.
func (mr *MockCronBackupReaderExMockRecorder) ListCronBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackupEx", reflect.TypeOf((*MockCronBackupReaderEx)(nil).ListCronBackupEx), ctx, query)
}

// MockCronBackupReader is a mock of CronBackupReader interface.
type MockCronBackupReader struct {
	ctrl     *gomock.Controller
	recorder *MockCronBackupReaderMockRecorder
}

// MockCronBackupReaderMockRecorder is the mock recorder for MockCronBackupReader.
type MockCronBackupReaderMockRecorder struct {
	mock *MockCronBackupReader
}

// NewMockCronBackupReader creates a new mock instance.
func NewMockCronBackupReader(ctrl *gomock.Controller) *MockCronBackupReader {
	mock := &MockCronBackupReader{ctrl: ctrl}
	mock.recorder = &MockCronBackupReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCronBackupReader) EXPECT() *MockCronBackupReaderMockRecorder {
	return m.recorder
}

// GetCronBackup mocks base method.
func (m *MockCronBackupReader) GetCronBackup(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackup", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackup indicates an expected call of GetCronBackup.
func (mr *MockCronBackupReaderMockRecorder) GetCronBackup(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackup", reflect.TypeOf((*MockCronBackupReader)(nil).GetCronBackup), ctx, name, resourceVersion)
}

// GetCronBackupEx mocks base method.
func (m *MockCronBackupReader) GetCronBackupEx(ctx context.Context, name, resourceVersion string) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCronBackupEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCronBackupEx indicates an expected call of GetCronBackupEx.
func (mr *MockCronBackupReaderMockRecorder) GetCronBackupEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCronBackupEx", reflect.TypeOf((*MockCronBackupReader)(nil).GetCronBackupEx), ctx, name, resourceVersion)
}

// ListCronBackupEx mocks base method.
func (m *MockCronBackupReader) ListCronBackupEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackupEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackupEx indicates an expected call of ListCronBackupEx.
func (mr *MockCronBackupReaderMockRecorder) ListCronBackupEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackupEx", reflect.TypeOf((*MockCronBackupReader)(nil).ListCronBackupEx), ctx, query)
}

// ListCronBackups mocks base method.
func (m *MockCronBackupReader) ListCronBackups(ctx context.Context, query *query.Query) (*v1.CronBackupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCronBackups", ctx, query)
	ret0, _ := ret[0].(*v1.CronBackupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCronBackups indicates an expected call of ListCronBackups.
func (mr *MockCronBackupReaderMockRecorder) ListCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCronBackups", reflect.TypeOf((*MockCronBackupReader)(nil).ListCronBackups), ctx, query)
}

// WatchCronBackups mocks base method.
func (m *MockCronBackupReader) WatchCronBackups(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCronBackups", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCronBackups indicates an expected call of WatchCronBackups.
func (mr *MockCronBackupReaderMockRecorder) WatchCronBackups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCronBackups", reflect.TypeOf((*MockCronBackupReader)(nil).WatchCronBackups), ctx, query)
}

// MockCronBackupWriter is a mock of CronBackupWriter interface.
type MockCronBackupWriter struct {
	ctrl     *gomock.Controller
	recorder *MockCronBackupWriterMockRecorder
}

// MockCronBackupWriterMockRecorder is the mock recorder for MockCronBackupWriter.
type MockCronBackupWriterMockRecorder struct {
	mock *MockCronBackupWriter
}

// NewMockCronBackupWriter creates a new mock instance.
func NewMockCronBackupWriter(ctrl *gomock.Controller) *MockCronBackupWriter {
	mock := &MockCronBackupWriter{ctrl: ctrl}
	mock.recorder = &MockCronBackupWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCronBackupWriter) EXPECT() *MockCronBackupWriterMockRecorder {
	return m.recorder
}

// CreateCronBackup mocks base method.
func (m *MockCronBackupWriter) CreateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCronBackup indicates an expected call of CreateCronBackup.
func (mr *MockCronBackupWriterMockRecorder) CreateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCronBackup", reflect.TypeOf((*MockCronBackupWriter)(nil).CreateCronBackup), ctx, cronBackup)
}

// DeleteCronBackup mocks base method.
func (m *MockCronBackupWriter) DeleteCronBackup(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackup", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackup indicates an expected call of DeleteCronBackup.
func (mr *MockCronBackupWriterMockRecorder) DeleteCronBackup(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackup", reflect.TypeOf((*MockCronBackupWriter)(nil).DeleteCronBackup), ctx, name)
}

// DeleteCronBackupCollection mocks base method.
func (m *MockCronBackupWriter) DeleteCronBackupCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCronBackupCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCronBackupCollection indicates an expected call of DeleteCronBackupCollection.
func (mr *MockCronBackupWriterMockRecorder) DeleteCronBackupCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCronBackupCollection", reflect.TypeOf((*MockCronBackupWriter)(nil).DeleteCronBackupCollection), ctx, query)
}

// UpdateCronBackup mocks base method.
func (m *MockCronBackupWriter) UpdateCronBackup(ctx context.Context, cronBackup *v1.CronBackup) (*v1.CronBackup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCronBackup", ctx, cronBackup)
	ret0, _ := ret[0].(*v1.CronBackup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCronBackup indicates an expected call of UpdateCronBackup.
func (mr *MockCronBackupWriterMockRecorder) UpdateCronBackup(ctx, cronBackup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCronBackup", reflect.TypeOf((*MockCronBackupWriter)(nil).UpdateCronBackup), ctx, cronBackup)
}

// MockDNSReader is a mock of DNSReader interface.
type MockDNSReader struct {
	ctrl     *gomock.Controller
	recorder *MockDNSReaderMockRecorder
}

// MockDNSReaderMockRecorder is the mock recorder for MockDNSReader.
type MockDNSReaderMockRecorder struct {
	mock *MockDNSReader
}

// NewMockDNSReader creates a new mock instance.
func NewMockDNSReader(ctrl *gomock.Controller) *MockDNSReader {
	mock := &MockDNSReader{ctrl: ctrl}
	mock.recorder = &MockDNSReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDNSReader) EXPECT() *MockDNSReaderMockRecorder {
	return m.recorder
}

// GetDomain mocks base method.
func (m *MockDNSReader) GetDomain(ctx context.Context, name string) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomain", ctx, name)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomain indicates an expected call of GetDomain.
func (mr *MockDNSReaderMockRecorder) GetDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomain", reflect.TypeOf((*MockDNSReader)(nil).GetDomain), ctx, name)
}

// ListDomains mocks base method.
func (m *MockDNSReader) ListDomains(ctx context.Context, query *query.Query) (*v1.DomainList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomains", ctx, query)
	ret0, _ := ret[0].(*v1.DomainList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomains indicates an expected call of ListDomains.
func (mr *MockDNSReaderMockRecorder) ListDomains(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomains", reflect.TypeOf((*MockDNSReader)(nil).ListDomains), ctx, query)
}

// ListDomainsEx mocks base method.
func (m *MockDNSReader) ListDomainsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomainsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomainsEx indicates an expected call of ListDomainsEx.
func (mr *MockDNSReaderMockRecorder) ListDomainsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomainsEx", reflect.TypeOf((*MockDNSReader)(nil).ListDomainsEx), ctx, query)
}

// ListRecordsEx mocks base method.
func (m *MockDNSReader) ListRecordsEx(ctx context.Context, name, subdomain string, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecordsEx", ctx, name, subdomain, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecordsEx indicates an expected call of ListRecordsEx.
func (mr *MockDNSReaderMockRecorder) ListRecordsEx(ctx, name, subdomain, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecordsEx", reflect.TypeOf((*MockDNSReader)(nil).ListRecordsEx), ctx, name, subdomain, query)
}

// WatchDomain mocks base method.
func (m *MockDNSReader) WatchDomain(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchDomain", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchDomain indicates an expected call of WatchDomain.
func (mr *MockDNSReaderMockRecorder) WatchDomain(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchDomain", reflect.TypeOf((*MockDNSReader)(nil).WatchDomain), ctx, query)
}

// MockDNSReaderEx is a mock of DNSReaderEx interface.
type MockDNSReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockDNSReaderExMockRecorder
}

// MockDNSReaderExMockRecorder is the mock recorder for MockDNSReaderEx.
type MockDNSReaderExMockRecorder struct {
	mock *MockDNSReaderEx
}

// NewMockDNSReaderEx creates a new mock instance.
func NewMockDNSReaderEx(ctrl *gomock.Controller) *MockDNSReaderEx {
	mock := &MockDNSReaderEx{ctrl: ctrl}
	mock.recorder = &MockDNSReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDNSReaderEx) EXPECT() *MockDNSReaderExMockRecorder {
	return m.recorder
}

// ListDomainsEx mocks base method.
func (m *MockDNSReaderEx) ListDomainsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomainsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomainsEx indicates an expected call of ListDomainsEx.
func (mr *MockDNSReaderExMockRecorder) ListDomainsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomainsEx", reflect.TypeOf((*MockDNSReaderEx)(nil).ListDomainsEx), ctx, query)
}

// ListRecordsEx mocks base method.
func (m *MockDNSReaderEx) ListRecordsEx(ctx context.Context, name, subdomain string, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecordsEx", ctx, name, subdomain, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecordsEx indicates an expected call of ListRecordsEx.
func (mr *MockDNSReaderExMockRecorder) ListRecordsEx(ctx, name, subdomain, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecordsEx", reflect.TypeOf((*MockDNSReaderEx)(nil).ListRecordsEx), ctx, name, subdomain, query)
}

// MockDNSWriter is a mock of DNSWriter interface.
type MockDNSWriter struct {
	ctrl     *gomock.Controller
	recorder *MockDNSWriterMockRecorder
}

// MockDNSWriterMockRecorder is the mock recorder for MockDNSWriter.
type MockDNSWriterMockRecorder struct {
	mock *MockDNSWriter
}

// NewMockDNSWriter creates a new mock instance.
func NewMockDNSWriter(ctrl *gomock.Controller) *MockDNSWriter {
	mock := &MockDNSWriter{ctrl: ctrl}
	mock.recorder = &MockDNSWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDNSWriter) EXPECT() *MockDNSWriterMockRecorder {
	return m.recorder
}

// CreateDomain mocks base method.
func (m *MockDNSWriter) CreateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDomain indicates an expected call of CreateDomain.
func (mr *MockDNSWriterMockRecorder) CreateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDomain", reflect.TypeOf((*MockDNSWriter)(nil).CreateDomain), ctx, domain)
}

// DeleteDomain mocks base method.
func (m *MockDNSWriter) DeleteDomain(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDomain", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDomain indicates an expected call of DeleteDomain.
func (mr *MockDNSWriterMockRecorder) DeleteDomain(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDomain", reflect.TypeOf((*MockDNSWriter)(nil).DeleteDomain), ctx, name)
}

// UpdateDomain mocks base method.
func (m *MockDNSWriter) UpdateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDomain", ctx, domain)
	ret0, _ := ret[0].(*v1.Domain)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDomain indicates an expected call of UpdateDomain.
func (mr *MockDNSWriterMockRecorder) UpdateDomain(ctx, domain interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDomain", reflect.TypeOf((*MockDNSWriter)(nil).UpdateDomain), ctx, domain)
}

// MockTemplateReader is a mock of TemplateReader interface.
type MockTemplateReader struct {
	ctrl     *gomock.Controller
	recorder *MockTemplateReaderMockRecorder
}

// MockTemplateReaderMockRecorder is the mock recorder for MockTemplateReader.
type MockTemplateReaderMockRecorder struct {
	mock *MockTemplateReader
}

// NewMockTemplateReader creates a new mock instance.
func NewMockTemplateReader(ctrl *gomock.Controller) *MockTemplateReader {
	mock := &MockTemplateReader{ctrl: ctrl}
	mock.recorder = &MockTemplateReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTemplateReader) EXPECT() *MockTemplateReaderMockRecorder {
	return m.recorder
}

// GetTemplate mocks base method.
func (m *MockTemplateReader) GetTemplate(ctx context.Context, name string) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplate", ctx, name)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplate indicates an expected call of GetTemplate.
func (mr *MockTemplateReaderMockRecorder) GetTemplate(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplate", reflect.TypeOf((*MockTemplateReader)(nil).GetTemplate), ctx, name)
}

// GetTemplateEx mocks base method.
func (m *MockTemplateReader) GetTemplateEx(ctx context.Context, name, resourceVersion string) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplateEx indicates an expected call of GetTemplateEx.
func (mr *MockTemplateReaderMockRecorder) GetTemplateEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateEx", reflect.TypeOf((*MockTemplateReader)(nil).GetTemplateEx), ctx, name, resourceVersion)
}

// ListTemplates mocks base method.
func (m *MockTemplateReader) ListTemplates(ctx context.Context, query *query.Query) (*v1.TemplateList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplates", ctx, query)
	ret0, _ := ret[0].(*v1.TemplateList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTemplates indicates an expected call of ListTemplates.
func (mr *MockTemplateReaderMockRecorder) ListTemplates(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplates", reflect.TypeOf((*MockTemplateReader)(nil).ListTemplates), ctx, query)
}

// ListTemplatesEx mocks base method.
func (m *MockTemplateReader) ListTemplatesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplatesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTemplatesEx indicates an expected call of ListTemplatesEx.
func (mr *MockTemplateReaderMockRecorder) ListTemplatesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplatesEx", reflect.TypeOf((*MockTemplateReader)(nil).ListTemplatesEx), ctx, query)
}

// WatchTemplates mocks base method.
func (m *MockTemplateReader) WatchTemplates(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchTemplates", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchTemplates indicates an expected call of WatchTemplates.
func (mr *MockTemplateReaderMockRecorder) WatchTemplates(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchTemplates", reflect.TypeOf((*MockTemplateReader)(nil).WatchTemplates), ctx, query)
}

// MockTemplateReaderEx is a mock of TemplateReaderEx interface.
type MockTemplateReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockTemplateReaderExMockRecorder
}

// MockTemplateReaderExMockRecorder is the mock recorder for MockTemplateReaderEx.
type MockTemplateReaderExMockRecorder struct {
	mock *MockTemplateReaderEx
}

// NewMockTemplateReaderEx creates a new mock instance.
func NewMockTemplateReaderEx(ctrl *gomock.Controller) *MockTemplateReaderEx {
	mock := &MockTemplateReaderEx{ctrl: ctrl}
	mock.recorder = &MockTemplateReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTemplateReaderEx) EXPECT() *MockTemplateReaderExMockRecorder {
	return m.recorder
}

// GetTemplateEx mocks base method.
func (m *MockTemplateReaderEx) GetTemplateEx(ctx context.Context, name, resourceVersion string) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplateEx indicates an expected call of GetTemplateEx.
func (mr *MockTemplateReaderExMockRecorder) GetTemplateEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateEx", reflect.TypeOf((*MockTemplateReaderEx)(nil).GetTemplateEx), ctx, name, resourceVersion)
}

// ListTemplatesEx mocks base method.
func (m *MockTemplateReaderEx) ListTemplatesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplatesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTemplatesEx indicates an expected call of ListTemplatesEx.
func (mr *MockTemplateReaderExMockRecorder) ListTemplatesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplatesEx", reflect.TypeOf((*MockTemplateReaderEx)(nil).ListTemplatesEx), ctx, query)
}

// MockTemplateWriter is a mock of TemplateWriter interface.
type MockTemplateWriter struct {
	ctrl     *gomock.Controller
	recorder *MockTemplateWriterMockRecorder
}

// MockTemplateWriterMockRecorder is the mock recorder for MockTemplateWriter.
type MockTemplateWriterMockRecorder struct {
	mock *MockTemplateWriter
}

// NewMockTemplateWriter creates a new mock instance.
func NewMockTemplateWriter(ctrl *gomock.Controller) *MockTemplateWriter {
	mock := &MockTemplateWriter{ctrl: ctrl}
	mock.recorder = &MockTemplateWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTemplateWriter) EXPECT() *MockTemplateWriterMockRecorder {
	return m.recorder
}

// CreateTemplate mocks base method.
func (m *MockTemplateWriter) CreateTemplate(ctx context.Context, template *v1.Template) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTemplate", ctx, template)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTemplate indicates an expected call of CreateTemplate.
func (mr *MockTemplateWriterMockRecorder) CreateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTemplate", reflect.TypeOf((*MockTemplateWriter)(nil).CreateTemplate), ctx, template)
}

// DeleteTemplate mocks base method.
func (m *MockTemplateWriter) DeleteTemplate(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTemplate", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTemplate indicates an expected call of DeleteTemplate.
func (mr *MockTemplateWriterMockRecorder) DeleteTemplate(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTemplate", reflect.TypeOf((*MockTemplateWriter)(nil).DeleteTemplate), ctx, name)
}

// DeleteTemplateCollection mocks base method.
func (m *MockTemplateWriter) DeleteTemplateCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTemplateCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTemplateCollection indicates an expected call of DeleteTemplateCollection.
func (mr *MockTemplateWriterMockRecorder) DeleteTemplateCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTemplateCollection", reflect.TypeOf((*MockTemplateWriter)(nil).DeleteTemplateCollection), ctx, query)
}

// UpdateTemplate mocks base method.
func (m *MockTemplateWriter) UpdateTemplate(ctx context.Context, template *v1.Template) (*v1.Template, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTemplate", ctx, template)
	ret0, _ := ret[0].(*v1.Template)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateTemplate indicates an expected call of UpdateTemplate.
func (mr *MockTemplateWriterMockRecorder) UpdateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTemplate", reflect.TypeOf((*MockTemplateWriter)(nil).UpdateTemplate), ctx, template)
}

// MockCloudProviderReader is a mock of CloudProviderReader interface.
type MockCloudProviderReader struct {
	ctrl     *gomock.Controller
	recorder *MockCloudProviderReaderMockRecorder
}

// MockCloudProviderReaderMockRecorder is the mock recorder for MockCloudProviderReader.
type MockCloudProviderReaderMockRecorder struct {
	mock *MockCloudProviderReader
}

// NewMockCloudProviderReader creates a new mock instance.
func NewMockCloudProviderReader(ctrl *gomock.Controller) *MockCloudProviderReader {
	mock := &MockCloudProviderReader{ctrl: ctrl}
	mock.recorder = &MockCloudProviderReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCloudProviderReader) EXPECT() *MockCloudProviderReaderMockRecorder {
	return m.recorder
}

// GetCloudProvider mocks base method.
func (m *MockCloudProviderReader) GetCloudProvider(ctx context.Context, name string) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudProvider", ctx, name)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCloudProvider indicates an expected call of GetCloudProvider.
func (mr *MockCloudProviderReaderMockRecorder) GetCloudProvider(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudProvider", reflect.TypeOf((*MockCloudProviderReader)(nil).GetCloudProvider), ctx, name)
}

// GetCloudProviderEx mocks base method.
func (m *MockCloudProviderReader) GetCloudProviderEx(ctx context.Context, name, resourceVersion string) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudProviderEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCloudProviderEx indicates an expected call of GetCloudProviderEx.
func (mr *MockCloudProviderReaderMockRecorder) GetCloudProviderEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudProviderEx", reflect.TypeOf((*MockCloudProviderReader)(nil).GetCloudProviderEx), ctx, name, resourceVersion)
}

// ListCloudProviders mocks base method.
func (m *MockCloudProviderReader) ListCloudProviders(ctx context.Context, query *query.Query) (*v1.CloudProviderList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCloudProviders", ctx, query)
	ret0, _ := ret[0].(*v1.CloudProviderList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCloudProviders indicates an expected call of ListCloudProviders.
func (mr *MockCloudProviderReaderMockRecorder) ListCloudProviders(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCloudProviders", reflect.TypeOf((*MockCloudProviderReader)(nil).ListCloudProviders), ctx, query)
}

// ListCloudProvidersEx mocks base method.
func (m *MockCloudProviderReader) ListCloudProvidersEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCloudProvidersEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCloudProvidersEx indicates an expected call of ListCloudProvidersEx.
func (mr *MockCloudProviderReaderMockRecorder) ListCloudProvidersEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCloudProvidersEx", reflect.TypeOf((*MockCloudProviderReader)(nil).ListCloudProvidersEx), ctx, query)
}

// WatchCloudProviders mocks base method.
func (m *MockCloudProviderReader) WatchCloudProviders(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCloudProviders", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCloudProviders indicates an expected call of WatchCloudProviders.
func (mr *MockCloudProviderReaderMockRecorder) WatchCloudProviders(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCloudProviders", reflect.TypeOf((*MockCloudProviderReader)(nil).WatchCloudProviders), ctx, query)
}

// MockCloudProviderEx is a mock of CloudProviderEx interface.
type MockCloudProviderEx struct {
	ctrl     *gomock.Controller
	recorder *MockCloudProviderExMockRecorder
}

// MockCloudProviderExMockRecorder is the mock recorder for MockCloudProviderEx.
type MockCloudProviderExMockRecorder struct {
	mock *MockCloudProviderEx
}

// NewMockCloudProviderEx creates a new mock instance.
func NewMockCloudProviderEx(ctrl *gomock.Controller) *MockCloudProviderEx {
	mock := &MockCloudProviderEx{ctrl: ctrl}
	mock.recorder = &MockCloudProviderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCloudProviderEx) EXPECT() *MockCloudProviderExMockRecorder {
	return m.recorder
}

// GetCloudProviderEx mocks base method.
func (m *MockCloudProviderEx) GetCloudProviderEx(ctx context.Context, name, resourceVersion string) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudProviderEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCloudProviderEx indicates an expected call of GetCloudProviderEx.
func (mr *MockCloudProviderExMockRecorder) GetCloudProviderEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudProviderEx", reflect.TypeOf((*MockCloudProviderEx)(nil).GetCloudProviderEx), ctx, name, resourceVersion)
}

// ListCloudProvidersEx mocks base method.
func (m *MockCloudProviderEx) ListCloudProvidersEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCloudProvidersEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCloudProvidersEx indicates an expected call of ListCloudProvidersEx.
func (mr *MockCloudProviderExMockRecorder) ListCloudProvidersEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCloudProvidersEx", reflect.TypeOf((*MockCloudProviderEx)(nil).ListCloudProvidersEx), ctx, query)
}

// MockCloudProviderWriter is a mock of CloudProviderWriter interface.
type MockCloudProviderWriter struct {
	ctrl     *gomock.Controller
	recorder *MockCloudProviderWriterMockRecorder
}

// MockCloudProviderWriterMockRecorder is the mock recorder for MockCloudProviderWriter.
type MockCloudProviderWriterMockRecorder struct {
	mock *MockCloudProviderWriter
}

// NewMockCloudProviderWriter creates a new mock instance.
func NewMockCloudProviderWriter(ctrl *gomock.Controller) *MockCloudProviderWriter {
	mock := &MockCloudProviderWriter{ctrl: ctrl}
	mock.recorder = &MockCloudProviderWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCloudProviderWriter) EXPECT() *MockCloudProviderWriterMockRecorder {
	return m.recorder
}

// CreateCloudProvider mocks base method.
func (m *MockCloudProviderWriter) CreateCloudProvider(ctx context.Context, cp *v1.CloudProvider) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCloudProvider", ctx, cp)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCloudProvider indicates an expected call of CreateCloudProvider.
func (mr *MockCloudProviderWriterMockRecorder) CreateCloudProvider(ctx, cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCloudProvider", reflect.TypeOf((*MockCloudProviderWriter)(nil).CreateCloudProvider), ctx, cp)
}

// DeleteCloudProvider mocks base method.
func (m *MockCloudProviderWriter) DeleteCloudProvider(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCloudProvider", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCloudProvider indicates an expected call of DeleteCloudProvider.
func (mr *MockCloudProviderWriterMockRecorder) DeleteCloudProvider(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCloudProvider", reflect.TypeOf((*MockCloudProviderWriter)(nil).DeleteCloudProvider), ctx, name)
}

// UpdateCloudProvider mocks base method.
func (m *MockCloudProviderWriter) UpdateCloudProvider(ctx context.Context, cp *v1.CloudProvider) (*v1.CloudProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCloudProvider", ctx, cp)
	ret0, _ := ret[0].(*v1.CloudProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCloudProvider indicates an expected call of UpdateCloudProvider.
func (mr *MockCloudProviderWriterMockRecorder) UpdateCloudProvider(ctx, cp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCloudProvider", reflect.TypeOf((*MockCloudProviderWriter)(nil).UpdateCloudProvider), ctx, cp)
}

// MockRegistryReader is a mock of RegistryReader interface.
type MockRegistryReader struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryReaderMockRecorder
}

// MockRegistryReaderMockRecorder is the mock recorder for MockRegistryReader.
type MockRegistryReaderMockRecorder struct {
	mock *MockRegistryReader
}

// NewMockRegistryReader creates a new mock instance.
func NewMockRegistryReader(ctrl *gomock.Controller) *MockRegistryReader {
	mock := &MockRegistryReader{ctrl: ctrl}
	mock.recorder = &MockRegistryReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistryReader) EXPECT() *MockRegistryReaderMockRecorder {
	return m.recorder
}

// GetRegistry mocks base method.
func (m *MockRegistryReader) GetRegistry(ctx context.Context, name string) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegistry", ctx, name)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegistry indicates an expected call of GetRegistry.
func (mr *MockRegistryReaderMockRecorder) GetRegistry(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegistry", reflect.TypeOf((*MockRegistryReader)(nil).GetRegistry), ctx, name)
}

// GetRegistryEx mocks base method.
func (m *MockRegistryReader) GetRegistryEx(ctx context.Context, name, resourceVersion string) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegistryEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegistryEx indicates an expected call of GetRegistryEx.
func (mr *MockRegistryReaderMockRecorder) GetRegistryEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegistryEx", reflect.TypeOf((*MockRegistryReader)(nil).GetRegistryEx), ctx, name, resourceVersion)
}

// ListRegistries mocks base method.
func (m *MockRegistryReader) ListRegistries(ctx context.Context, query *query.Query) (*v1.RegistryList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegistries", ctx, query)
	ret0, _ := ret[0].(*v1.RegistryList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegistries indicates an expected call of ListRegistries.
func (mr *MockRegistryReaderMockRecorder) ListRegistries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegistries", reflect.TypeOf((*MockRegistryReader)(nil).ListRegistries), ctx, query)
}

// ListRegistriesEx mocks base method.
func (m *MockRegistryReader) ListRegistriesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegistriesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegistriesEx indicates an expected call of ListRegistriesEx.
func (mr *MockRegistryReaderMockRecorder) ListRegistriesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegistriesEx", reflect.TypeOf((*MockRegistryReader)(nil).ListRegistriesEx), ctx, query)
}

// WatchRegistries mocks base method.
func (m *MockRegistryReader) WatchRegistries(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRegistries", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRegistries indicates an expected call of WatchRegistries.
func (mr *MockRegistryReaderMockRecorder) WatchRegistries(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRegistries", reflect.TypeOf((*MockRegistryReader)(nil).WatchRegistries), ctx, query)
}

// MockRegistryEx is a mock of RegistryEx interface.
type MockRegistryEx struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryExMockRecorder
}

// MockRegistryExMockRecorder is the mock recorder for MockRegistryEx.
type MockRegistryExMockRecorder struct {
	mock *MockRegistryEx
}

// NewMockRegistryEx creates a new mock instance.
func NewMockRegistryEx(ctrl *gomock.Controller) *MockRegistryEx {
	mock := &MockRegistryEx{ctrl: ctrl}
	mock.recorder = &MockRegistryExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistryEx) EXPECT() *MockRegistryExMockRecorder {
	return m.recorder
}

// GetRegistryEx mocks base method.
func (m *MockRegistryEx) GetRegistryEx(ctx context.Context, name, resourceVersion string) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegistryEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegistryEx indicates an expected call of GetRegistryEx.
func (mr *MockRegistryExMockRecorder) GetRegistryEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegistryEx", reflect.TypeOf((*MockRegistryEx)(nil).GetRegistryEx), ctx, name, resourceVersion)
}

// ListRegistriesEx mocks base method.
func (m *MockRegistryEx) ListRegistriesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRegistriesEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegistriesEx indicates an expected call of ListRegistriesEx.
func (mr *MockRegistryExMockRecorder) ListRegistriesEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegistriesEx", reflect.TypeOf((*MockRegistryEx)(nil).ListRegistriesEx), ctx, query)
}

// MockRegistryWriter is a mock of RegistryWriter interface.
type MockRegistryWriter struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryWriterMockRecorder
}

// MockRegistryWriterMockRecorder is the mock recorder for MockRegistryWriter.
type MockRegistryWriterMockRecorder struct {
	mock *MockRegistryWriter
}

// NewMockRegistryWriter creates a new mock instance.
func NewMockRegistryWriter(ctrl *gomock.Controller) *MockRegistryWriter {
	mock := &MockRegistryWriter{ctrl: ctrl}
	mock.recorder = &MockRegistryWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistryWriter) EXPECT() *MockRegistryWriterMockRecorder {
	return m.recorder
}

// CreateRegistry mocks base method.
func (m *MockRegistryWriter) CreateRegistry(ctx context.Context, r *v1.Registry) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRegistry", ctx, r)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRegistry indicates an expected call of CreateRegistry.
func (mr *MockRegistryWriterMockRecorder) CreateRegistry(ctx, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRegistry", reflect.TypeOf((*MockRegistryWriter)(nil).CreateRegistry), ctx, r)
}

// DeleteRegistry mocks base method.
func (m *MockRegistryWriter) DeleteRegistry(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRegistry", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRegistry indicates an expected call of DeleteRegistry.
func (mr *MockRegistryWriterMockRecorder) DeleteRegistry(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRegistry", reflect.TypeOf((*MockRegistryWriter)(nil).DeleteRegistry), ctx, name)
}

// UpdateRegistry mocks base method.
func (m *MockRegistryWriter) UpdateRegistry(ctx context.Context, r *v1.Registry) (*v1.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRegistry", ctx, r)
	ret0, _ := ret[0].(*v1.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRegistry indicates an expected call of UpdateRegistry.
func (mr *MockRegistryWriterMockRecorder) UpdateRegistry(ctx, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRegistry", reflect.TypeOf((*MockRegistryWriter)(nil).UpdateRegistry), ctx, r)
}
