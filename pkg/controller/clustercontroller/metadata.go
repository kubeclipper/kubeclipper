package clustercontroller

import (
	"context"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// assembleClusterExtraMetadata assemble cluster extra metadata, used to share data between the various steps
func (r *ClusterReconciler) assembleClusterExtraMetadata(ctx context.Context, c *v1.Cluster) (*component.ExtraMetadata, error) {
	meta := &component.ExtraMetadata{
		ClusterName:        c.Name,
		ClusterStatus:      c.Status.Phase,
		Offline:            c.Offline(),
		LocalRegistry:      c.LocalRegistry,
		CRI:                c.ContainerRuntime.Type,
		KubeVersion:        c.KubernetesVersion,
		KubeletDataDir:     c.Kubelet.RootDir,
		ControlPlaneStatus: c.Status.ControlPlaneHealth,
		CNI:                c.CNI.Type,
		CNINamespace:       c.CNI.Namespace,
	}
	meta.Addons = append(meta.Addons, c.Addons...)

	masters, err := r.convertNodes(ctx, c.Masters)
	if err != nil {
		return nil, err
	}
	meta.Masters = append(meta.Masters, masters...)
	workers, err := r.convertNodes(ctx, c.Workers)
	if err != nil {
		return nil, err
	}
	meta.Workers = append(meta.Workers, workers...)
	return meta, nil
}

// convertNodes convert nodes information, this is especially important for handling node information in subsequent operations
func (r *ClusterReconciler) convertNodes(ctx context.Context, nodes v1.WorkerNodeList) ([]component.Node, error) {
	var meta []component.Node
	for _, node := range nodes {
		n, err := r.ClusterOperator.GetNodeEx(ctx, node.ID, "0")
		if err != nil {
			return nil, err
		}
		item := component.Node{
			ID:       n.Name,
			IPv4:     n.Status.Ipv4DefaultIP,
			NodeIPv4: n.Status.NodeIpv4DefaultIP,
			Region:   n.Labels[common.LabelTopologyRegion],
			Hostname: n.Labels[common.LabelHostname],
			Role:     n.Labels[common.LabelNodeRole],
		}
		_, item.Disable = n.Labels[common.LabelNodeDisable]
		meta = append(meta, item)
	}

	return meta, nil
}
