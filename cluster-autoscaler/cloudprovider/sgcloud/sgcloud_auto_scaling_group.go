/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sgcloud

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/klog/v2"
	schedulerframework "k8s.io/kubernetes/pkg/scheduler/framework"
)

// AutoScalingGroup implements NodeGroup interface
type AutoScalingGroup struct {
	manager   *sgcloudManager
	groupID   string
	groupName string
	regionID  string
	minSize   int
	maxSize   int
}

// Check if our AutoScalingGroup implements necessary interface.
var _ cloudprovider.NodeGroup = &AutoScalingGroup{}

// MaxSize returns maximum size of the node group.
func (asg *AutoScalingGroup) MaxSize() int {
	return asg.maxSize
}

// MinSize returns minimum size of the node group.
func (asg *AutoScalingGroup) MinSize() int {
	return asg.minSize
}

// TargetSize returns the current target size of the node group. It is possible that the
// number of nodes in Kubernetes is different at the moment but should be equal
// to Size() once everything stabilizes (new nodes finish startup and registration or
// removed nodes are deleted completely). Implementation required.
//
// Target size is desire instance number of the auto scaling group, and not equal to current instance number if the
// auto scaling group is in increasing or decreasing process.
func (asg *AutoScalingGroup) TargetSize() (int, error) {
	desireNumber, err := asg.manager.GetAsgSize(asg)
	if err != nil {
		klog.Warningf("failed to get group target size. groupID: %s, error: %v", asg.groupID, err)
		return 0, err
	}

	return int(desireNumber), nil
}

// IncreaseSize increases the size of the node group. To delete a node you need
// to explicitly name it and use DeleteNode. This function should wait until
// node group size is updated. Implementation required.
func (asg *AutoScalingGroup) IncreaseSize(delta int) error {
	klog.Infof("increase ASG:%s with %d nodes", asg.Id(), delta)
	if delta <= 0 {
		return fmt.Errorf("size increase must be positive")
	}
	size, err := asg.manager.GetAsgSize(asg)
	if err != nil {
		klog.Errorf("failed to get ASG size because of %s", err.Error())
		return err
	}
	if int(size)+delta > asg.MaxSize() {
		return fmt.Errorf("size increase is too large - desired:%d max:%d", int(size)+delta, asg.MaxSize())
	}
	return asg.manager.AddInstances(asg, delta)
}

// Belongs returns true if the given node belongs to the NodeGroup.
func (asg *AutoScalingGroup) Belongs(node *apiv1.Node) (bool, error) {
	instanceId, err := getInstanceIdFromProviderId(node.Spec.ProviderID)
	if err != nil {
		return false, err
	}
	targetAsg, err := asg.manager.GetAsgForInstance(instanceId)
	if err != nil {
		return false, err
	}
	if targetAsg == nil {
		return false, fmt.Errorf("%s doesn't belong to a known Asg", node.Name)
	}
	if targetAsg.Id() != asg.Id() {
		return false, nil
	}
	return true, nil
}

// DeleteNodes deletes nodes from this node group. Error is returned either on
// failure or if the given node doesn't belong to this node group. This function
// should wait until node group size is updated. Implementation required.
func (asg *AutoScalingGroup) DeleteNodes(nodes []*apiv1.Node) error {
	size, err := asg.manager.GetAsgSize(asg)
	if err != nil {
		klog.Errorf("failed to get ASG size because of %s", err.Error())
		return err
	}
	if int(size) <= asg.MinSize() {
		return fmt.Errorf("min size reached, nodes will not be deleted")
	}
	nodeIds := make([]string, 0, len(nodes))
	for _, node := range nodes {
		belongs, err := asg.Belongs(node)
		if err != nil {
			klog.Errorf("failed to check whether node:%s is belong to asg:%s", node.GetName(), asg.Id())
			return err
		}
		if !belongs {
			return fmt.Errorf("%s belongs to a different asg than %s", node.Name, asg.Id())
		}
		instanceId, err := getInstanceIdFromProviderId(node.Spec.ProviderID)
		if err != nil {
			klog.Errorf("failed to find instanceId from providerId,because of %s", err.Error())
			return err
		}
		nodeIds = append(nodeIds, instanceId)
	}
	return asg.manager.DeleteInstances(nodeIds)
}

// DecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the
// request for new nodes that have not been yet fulfilled. Delta should be negative.
// It is assumed that cloud provider will not delete the existing nodes when there
// is an option to just decrease the target. Implementation required.
func (asg *AutoScalingGroup) DecreaseTargetSize(delta int) error {
	// klog.V(4).Infof("Aliyun: DecreaseTargetSize() with args: %v", delta)
	// if delta >= 0 {
	// 	return fmt.Errorf("size decrease size must be negative")
	// }
	// size, err := asg.manager.GetAsgSize(asg)
	// if err != nil {
	// 	klog.Errorf("failed to get ASG size because of %s", err.Error())
	// 	return err
	// }
	// nodes, err := asg.manager.GetAsgNodes(asg)
	// if err != nil {
	// 	klog.Errorf("failed to get ASG nodes because of %s", err.Error())
	// 	return err
	// }
	// if int(size)+delta < len(nodes) {
	// 	return fmt.Errorf("attempt to delete existing nodes targetSize:%d delta:%d existingNodes: %d",
	// 		size, delta, len(nodes))
	// }
	// return asg.manager.SetAsgSize(asg, size+int64(delta))
	return cloudprovider.ErrNotImplemented
}

// Id returns an unique identifier of the node group.
func (asg *AutoScalingGroup) Id() string {
	return asg.groupID
}

// RegionId returns regionId of asg
func (asg *AutoScalingGroup) RegionId() string {
	return asg.regionID
}

// Debug returns a debug string for the Asg.
func (asg *AutoScalingGroup) Debug() string {
	return fmt.Sprintf("%s (%d:%d)", asg.Id(), asg.MinSize(), asg.MaxSize())
}

// Nodes returns a list of all nodes that belong to this node group.
// It is required that Instance objects returned by this method have Id field set.
// Other fields are optional.
// This list should include also instances that might have not become a kubernetes node yet.
func (asg *AutoScalingGroup) Nodes() ([]cloudprovider.Instance, error) {
	instanceNames, err := asg.manager.GetAsgNodes(asg)
	if err != nil {
		return nil, err
	}
	instances := make([]cloudprovider.Instance, 0, len(instanceNames))
	for _, instanceName := range instanceNames {
		instances = append(instances, cloudprovider.Instance{Id: instanceName})
	}
	return instances, nil
}

// TemplateNodeInfo returns a schedulerframework.NodeInfo structure of an empty
// (as if just started) node. This will be used in scale-up simulations to
// predict what would a new node look like if a node group was expanded. The returned
// NodeInfo is expected to have a fully populated Node object, with all of the labels,
// capacity and allocatable information as well as all pods that are started on
// the node by default, using manifest (most likely only kube-proxy). Implementation optional.
func (asg *AutoScalingGroup) TemplateNodeInfo() (*schedulerframework.NodeInfo, error) {
	template, err := asg.manager.getAsgTmpl()
	if err != nil {
		return nil, err
	}

	node, err := asg.manager.buildNodeFromTemplate(asg, template)
	if err != nil {
		klog.Errorf("failed to build instanceType:%v from template in ASG:%s,because of %s", template.InstanceType, asg.Id(), err.Error())
		return nil, err
	}

	nodeInfo := schedulerframework.NewNodeInfo(cloudprovider.BuildKubeProxy(asg.groupID))
	nodeInfo.SetNode(node)
	return nodeInfo, nil
}

// Exist checks if the node group really exists on the cloud provider side. Allows to tell the
// theoretical node group from the real one. Implementation required.
func (asg *AutoScalingGroup) Exist() bool {
	// Since all group synced from remote and we do not support auto provision,
	// so we can assume that the group always exist.
	return true
}

// Create creates the node group on the cloud provider side. Implementation optional.
func (asg *AutoScalingGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// Delete deletes the node group on the cloud provider side.
// This will be executed only for autoprovisioned node groups, once their size drops to 0.
// Implementation optional.
func (asg *AutoScalingGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

// Autoprovisioned returns true if the node group is autoprovisioned. An autoprovisioned group
// was created by CA and can be deleted when scaled to 0.
//
// Always return false because the node group should maintained by user.
func (asg *AutoScalingGroup) Autoprovisioned() bool {
	return false
}

// GetOptions returns NodeGroupAutoscalingOptions that should be used for this particular
// NodeGroup. Returning a nil will result in using default options.
func (asg *AutoScalingGroup) GetOptions(defaults config.NodeGroupAutoscalingOptions) (*config.NodeGroupAutoscalingOptions, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// String dumps current groups meta data.
func (asg *AutoScalingGroup) String() string {
	return fmt.Sprintf("group: %s min=%d max=%d", asg.groupID, asg.minSize, asg.maxSize)
}
