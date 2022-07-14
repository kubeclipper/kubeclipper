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

package kc

import (
	"context"
	"encoding/json"
	"fmt"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	apimachineryversion "k8s.io/apimachinery/pkg/version"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	listNodesPath     = "/api/core.kubeclipper.io/v1/nodes"
	clustersPath      = "/api/core.kubeclipper.io/v1/clusters"
	usersPath         = "/api/iam.kubeclipper.io/v1/users"
	rolesPath         = "/api/iam.kubeclipper.io/v1/roles"
	platformPath      = "/api/config.kubeclipper.io/v1/template"
	versionPath       = "/version"
	componentMetaPath = "/api/config.kubeclipper.io/v1/componentmeta"
)

func (cli *Client) ListNodes(ctx context.Context, query Queries) (*NodesList, error) {
	serverResp, err := cli.get(ctx, listNodesPath, query.ToRawQuery(), nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	nodes := NodesList{}
	err = json.NewDecoder(serverResp.body).Decode(&nodes)
	return &nodes, err
}

func (cli *Client) DescribeNode(ctx context.Context, name string) (*NodesList, error) {
	serverResp, err := cli.get(ctx, fmt.Sprintf("%s/%s", listNodesPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	node := v1.Node{}
	err = json.NewDecoder(serverResp.body).Decode(&node)
	nodes := NodesList{
		Items: []v1.Node{node},
	}
	return &nodes, err
}

func (cli *Client) DeleteNode(ctx context.Context, name string) error {
	serverResp, err := cli.delete(ctx, fmt.Sprintf("%s/%s", listNodesPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	return err
}

func (cli *Client) ListUsers(ctx context.Context, query Queries) (*UsersList, error) {
	serverResp, err := cli.get(ctx, usersPath, query.ToRawQuery(), nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	users := UsersList{}
	err = json.NewDecoder(serverResp.body).Decode(&users)
	return &users, err
}

func (cli *Client) DescribeUser(ctx context.Context, name string) (*UsersList, error) {
	serverResp, err := cli.get(ctx, fmt.Sprintf("%s/%s", usersPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	user := iamv1.User{}
	err = json.NewDecoder(serverResp.body).Decode(&user)
	users := UsersList{
		Items: []iamv1.User{user},
	}
	return &users, err
}

func (cli *Client) ListClusters(ctx context.Context, query Queries) (*ClustersList, error) {
	serverResp, err := cli.get(ctx, clustersPath, query.ToRawQuery(), nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	users := ClustersList{}
	err = json.NewDecoder(serverResp.body).Decode(&users)
	return &users, err
}

func (cli *Client) DescribeCluster(ctx context.Context, name string) (*ClustersList, error) {
	serverResp, err := cli.get(ctx, fmt.Sprintf("%s/%s", clustersPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	cluster := v1.Cluster{}
	err = json.NewDecoder(serverResp.body).Decode(&cluster)
	clusters := ClustersList{
		Items: []v1.Cluster{cluster},
	}
	return &clusters, err
}

func (cli *Client) ListRoles(ctx context.Context, query Queries) (*RoleList, error) {
	serverResp, err := cli.get(ctx, rolesPath, query.ToRawQuery(), nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	roles := RoleList{}
	err = json.NewDecoder(serverResp.body).Decode(&roles)
	return &roles, err
}

func (cli *Client) DescribeRole(ctx context.Context, name string) (*RoleList, error) {
	serverResp, err := cli.get(ctx, fmt.Sprintf("%s/%s", rolesPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	role := iamv1.GlobalRole{}
	err = json.NewDecoder(serverResp.body).Decode(&role)
	roles := RoleList{
		Items: []iamv1.GlobalRole{role},
	}
	return &roles, err
}

func (cli *Client) Version(ctx context.Context) (*apimachineryversion.Info, error) {
	serverResp, err := cli.get(ctx, versionPath, nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := &apimachineryversion.Info{}
	err = json.NewDecoder(serverResp.body).Decode(v)
	return v, err
}

func (cli *Client) CreateCluster(ctx context.Context, cluster *v1.Cluster) (*ClustersList, error) {
	serverResp, err := cli.post(ctx, clustersPath, nil, cluster, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := v1.Cluster{}
	err = json.NewDecoder(serverResp.body).Decode(&v)
	clusters := ClustersList{
		Items: []v1.Cluster{v},
	}
	return &clusters, err
}

func (cli *Client) CreateUser(ctx context.Context, user *iamv1.User) (*UsersList, error) {
	serverResp, err := cli.post(ctx, usersPath, nil, user, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := iamv1.User{}
	err = json.NewDecoder(serverResp.body).Decode(&v)
	users := UsersList{
		Items: []iamv1.User{v},
	}
	return &users, err
}

func (cli *Client) CreateRole(ctx context.Context, role *iamv1.GlobalRole) (*RoleList, error) {
	serverResp, err := cli.post(ctx, rolesPath, nil, role, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := iamv1.GlobalRole{}
	err = json.NewDecoder(serverResp.body).Decode(&v)
	roles := RoleList{
		Items: []iamv1.GlobalRole{v},
	}
	return &roles, err
}

func (cli *Client) DeleteUser(ctx context.Context, name string) error {
	serverResp, err := cli.delete(ctx, fmt.Sprintf("%s/%s", usersPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return err
	}
	return nil
}

func (cli *Client) DeleteRole(ctx context.Context, name string) error {
	serverResp, err := cli.delete(ctx, fmt.Sprintf("%s/%s", rolesPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return err
	}
	return nil
}

func (cli *Client) DeleteCluster(ctx context.Context, name string) error {
	serverResp, err := cli.delete(ctx, fmt.Sprintf("%s/%s", clustersPath, name), nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return err
	}
	return nil
}

func (cli *Client) GetPlatformSetting(ctx context.Context) (*v1.DockerRegistry, error) {
	serverResp, err := cli.get(ctx, platformPath, nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := v1.DockerRegistry{}
	err = json.NewDecoder(serverResp.body).Decode(&v)
	return &v, err
}

func (cli *Client) GetComponentMeta(ctx context.Context) (*ComponentMeta, error) {
	serverResp, err := cli.get(ctx, componentMetaPath, nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return nil, err
	}
	v := ComponentMeta{}
	err = json.NewDecoder(serverResp.body).Decode(&v)
	return &v, err
}
