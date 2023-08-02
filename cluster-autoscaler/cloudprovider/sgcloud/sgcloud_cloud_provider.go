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
	"os"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/autoscaler/cluster-autoscaler/config/dynamic"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"
	"k8s.io/autoscaler/cluster-autoscaler/utils/gpu"
	klog "k8s.io/klog/v2"
)

var _ cloudprovider.CloudProvider = (*sgcloudCloudProvider)(nil)

const (
	// GPULabel is the label added to nodes with GPU resource.
	GPULabel = "sgcloud/nvidia_name"
)

var (
	availableGPUTypes = map[string]struct{}{
		"nTeslaV100":    {},
		"nTeslaP40":     {},
		"nTeslaP4":      {},
		"nTeslaV100-16": {},
		"nTeslaV100-32": {},
	}
)

// sgcloudCloudProvider implements CloudProvider interface defined in autoscaler/cluster-autoscaler/cloudprovider/cloud_provider.go
type sgcloudCloudProvider struct {
	manager          *sgcloudManager
	resourceLimiter  *cloudprovider.ResourceLimiter
	autoScalingGroup []*AutoScalingGroup
}

// BuildSgCloudProvider builds CloudProvider implementation for SgCloud.
func BuildSgCloudProvider(manager *sgcloudManager, discoveryOpts cloudprovider.NodeGroupDiscoveryOptions, resourceLimiter *cloudprovider.ResourceLimiter) (cloudprovider.CloudProvider, error) {
	// TODO add discoveryOpts parameters check.
	if discoveryOpts.StaticDiscoverySpecified() {
		return buildStaticallyDiscoveringProvider(manager, discoveryOpts.NodeGroupSpecs, resourceLimiter)
	}
	if discoveryOpts.AutoDiscoverySpecified() {
		return nil, fmt.Errorf("only support static discovery scaling group in alicloud for now")
	}
	return nil, fmt.Errorf("failed to build alicloud provider: node group specs must be specified")
}

func buildStaticallyDiscoveringProvider(manager *sgcloudManager, specs []string, resourceLimiter *cloudprovider.ResourceLimiter) (*sgcloudCloudProvider, error) {
	sgcp := &sgcloudCloudProvider{
		manager:          manager,
		autoScalingGroup: make([]*AutoScalingGroup, 0),
		resourceLimiter:  resourceLimiter,
	}
	for _, spec := range specs {
		if err := sgcp.addNodeGroup(spec); err != nil {
			klog.Warningf("failed to add node group to alicloud provider with spec: %s", spec)
			return nil, err
		}
	}
	return sgcp, nil
}

func (sg *sgcloudCloudProvider) addNodeGroup(spec string) error {
	asg, err := buildAsgFromSpec(spec, sg.manager)
	if err != nil {
		klog.Errorf("failed to build ASG from spec, because of %s", err.Error())
		return err
	}
	sg.addAsg(asg)
	return nil
}

func buildAsgFromSpec(specStr string, manager *sgcloudManager) (*AutoScalingGroup, error) {
	spec, err := dynamic.SpecFromString(specStr, true)

	if err != nil {
		return nil, fmt.Errorf("failed to parse node group spec: %v", err)
	}

	// check auto scaling group is exists or not
	// _, err = manager.service.getScalingGroupByID(spec.Name)
	// if err != nil {
	// 	klog.Errorf("your scaling group: %s does not exist", spec.Name)
	// 	return nil, err
	// }

	asg := buildAsg(manager, spec.MinSize, spec.MaxSize, spec.Name)

	return asg, nil
}

func (sg *sgcloudCloudProvider) addAsg(asg *AutoScalingGroup) {
	sg.autoScalingGroup = append(sg.autoScalingGroup, asg)
	sg.manager.RegisterAsg(asg)
}

// Name returns the name of the cloud provider.
func (sg *sgcloudCloudProvider) Name() string {
	return cloudprovider.SgCloudProviderName
}

// NodeGroups returns all node groups managed by this cloud provider.
func (sg *sgcloudCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	result := make([]cloudprovider.NodeGroup, 0, len(sg.autoScalingGroup))
	for _, asg := range sg.autoScalingGroup {
		result = append(result, asg)
	}
	return result
}

// NodeGroupForNode returns the node group for the given node, nil if the node
// should not be processed by cluster autoscaler, or non-nil error if such
// occurred. Must be implemented.
func (sg *sgcloudCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	instanceID, err := getInstanceIdFromProviderId(node.Spec.ProviderID)
	if err != nil {
		klog.Errorf("failed to get instance Id from provider Id:%s,because of %s", node.Spec.ProviderID, err.Error())
		return nil, err
	}
	if len(instanceID) == 0 {
		klog.Warningf("Node %v has no providerId", node.Name)
		return nil, fmt.Errorf("provider id missing from node: %s", node.Name)
	}

	return sg.manager.GetAsgForInstance(instanceID)
}

// HasInstance returns whether a given node has a corresponding instance in this cloud provider
func (sg *sgcloudCloudProvider) HasInstance(node *apiv1.Node) (bool, error) {
	return true, cloudprovider.ErrNotImplemented
}

// Pricing returns pricing model for this cloud provider or error if not available. Not implemented.
func (sg *sgcloudCloudProvider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetAvailableMachineTypes get all machine types that can be requested from the cloud provider. Not implemented.
func (sg *sgcloudCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return []string{}, nil
}

// NewNodeGroup builds a theoretical node group based on the node definition provided. The node group is not automatically
// created on the cloud provider side. The node group is not returned by NodeGroups() until it is created. Not implemented.
func (sg *sgcloudCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string,
	taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetResourceLimiter returns struct containing limits (max, min) for resources (cores, memory etc.).
func (sg *sgcloudCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return sg.resourceLimiter, nil
}

// GPULabel returns the label added to nodes with GPU resource.
func (sg *sgcloudCloudProvider) GPULabel() string {
	return GPULabel
}

// GetAvailableGPUTypes returns all available GPU types cloud provider supports.
func (sg *sgcloudCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return availableGPUTypes
}

// GetNodeGpuConfig returns the label, type and resource name for the GPU added to node. If node doesn't have
// any GPUs, it returns nil.
func (sg *sgcloudCloudProvider) GetNodeGpuConfig(node *apiv1.Node) *cloudprovider.GpuConfig {
	return gpu.GetNodeGPUFromCloudProvider(sg, node)
}

// Cleanup currently does nothing.
func (sg *sgcloudCloudProvider) Cleanup() error {
	return nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
// Currently does nothing.
func (sg *sgcloudCloudProvider) Refresh() error {
	return nil
}

// BuildSgcloud returns sgcloud provider
func BuildSgcloud(opts config.AutoscalingOptions, do cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	var sgManager *sgcloudManager
	var err error
	if opts.CloudConfig != "" {
		config, fileErr := os.Open(opts.CloudConfig)
		if fileErr != nil {
			klog.Fatalf("Couldn't open cloud provider configuration %s: %#v", opts.CloudConfig, fileErr)
		}
		defer config.Close()
		sgManager, err = CreateSgcloudManager(config)
	} else {
		sgManager, err = CreateSgcloudManager(nil)
	}
	if err != nil {
		klog.Fatalf("Failed to create sgcloud Manager: %v", err)
	}
	cloudProvider, err := BuildSgCloudProvider(sgManager, do, rl)
	if err != nil {
		klog.Fatalf("Failed to create sgcloud cloud provider: %v", err)
	}
	return cloudProvider
}

// getInstanceIdFromProviderId must be in format: `REGION.INSTANCE_ID`
func getInstanceIdFromProviderId(id string) (string, error) {
	parts := strings.Split(id, "//")
	if len(parts) < 2 {
		return "", fmt.Errorf("sgCloud: unexpected ProviderID format, providerID=%s", id)
	}
	return parts[1], nil
}

func buildAsg(manager *sgcloudManager, minSize int, maxSize int, name string) *AutoScalingGroup {
	return &AutoScalingGroup{
		manager: manager,
		minSize: minSize,
		maxSize: maxSize,
		groupID: name,
	}
}
