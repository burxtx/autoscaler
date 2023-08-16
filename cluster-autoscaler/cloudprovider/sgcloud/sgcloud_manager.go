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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/cc"

	"k8s.io/klog/v2"
)

const (
	sdkCoolDownTimeout     = 200 * time.Millisecond
	defaultPodAmountsLimit = 110
	//ResourceGPU GPU resource type
	ResourceGPU apiv1.ResourceName = "nvidia.com/gpu"
)

type asgInformation struct {
	config *AutoScalingGroup
}
type sgcloudManager struct {
	cloudConfig *CloudConfig
	asgCache    *autoScalingGroupCache
	service     *sgcloudWrapper
}

type asgTemplate struct {
	InstanceType     int
	Region           string
	Zone             string
	CPU              int
	Memory           int
	GpuCount         int
	EphemeralStorage int
	Tags             map[string]string
}

// CreateSgcloudManager constructs cloudmanager object.
func CreateSgcloudManager(configReader io.Reader) (*sgcloudManager, error) {
	cfg := &CloudConfig{}
	if configReader != nil {
		configContents, err := ioutil.ReadAll(configReader)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(configContents, cfg)
		if err != nil {
			return nil, err
		}
	}
	if cfg.validate() != nil {
		return nil, errors.New("please check whether you have provided correct AccessKeyId,AccessKeySecret,RegionId or STS Token")
	}

	sgcloud, err := NewsgcloudWrapper(cfg)
	if err != nil {
		klog.Errorf("failed to create NewAutoScalingWrapper because of %s", err.Error())
		return nil, err
	}

	manager := &sgcloudManager{
		cloudConfig: cfg,
		asgCache:    newAutoScalingGroups(sgcloud),
		service:     sgcloud,
	}
	return manager, nil
}

// RegisterAsg registers asg in Cloud Manager.
func (m *sgcloudManager) RegisterAsg(asg *AutoScalingGroup) {
	m.asgCache.Register(asg)
}

// GetAsgForInstance returns AsgConfig of the given Instance
func (m *sgcloudManager) GetAsgForInstance(instanceId string) (*AutoScalingGroup, error) {
	return m.asgCache.FindForInstance(instanceId)
}

// GetAsgSize gets ASG size.
func (m *sgcloudManager) GetAsgSize(asgConfig *AutoScalingGroup) (int64, error) {
	instances, err := m.service.getScalingInstancesByCluster(asgConfig.manager.cloudConfig.ClusterID)
	if err != nil {
		return -1, fmt.Errorf("failed to describe ASG %s,Because of %s", asgConfig.groupID, err.Error())
	}
	return int64(len(instances)), nil
}

// AddInstances sets ASG size.
func (m *sgcloudManager) AddInstances(asg *AutoScalingGroup, delta int) error {
	var args = &cc.AddInstanceArgs{
		ClusterID: m.cloudConfig.ClusterID,
		Delta:     delta,
	}
	err := m.service.IncreaseCapcityInstanceSize(args)
	if err != nil {
		return fmt.Errorf("ScaleUpCluster error: %v", err)
	}
	return nil
}

// DeleteInstances deletes the given instances. All instances must be controlled by the same ASG.
func (m *sgcloudManager) DeleteInstances(instanceIds []string) error {
	klog.Infof("start to remove Instances from ASG %v", instanceIds)
	if len(instanceIds) == 0 {
		klog.Warningf("you don't provide any instanceIds to remove")
		return nil
	}
	// Check whether instances are in the same group
	// TODO: remove or provide more meaningful check method.
	commonAsg, err := m.asgCache.FindForInstance(instanceIds[0])
	if err != nil {
		klog.Errorf("failed to find instance:%s in ASG", instanceIds[0])
		return err
	}
	for _, instanceId := range instanceIds {
		asg, err := m.asgCache.FindForInstance(instanceId)
		if err != nil {
			klog.Errorf("failed to find instanceId:%s from ASG and exit", instanceId)
			return err
		}
		if asg != commonAsg {
			return fmt.Errorf("cannot delete instances which doesn't belong to the same ASG")
		}
	}
	nodeinfos := make([]string, len(instanceIds))
	nodeinfos = append(nodeinfos, instanceIds...)

	args := &cc.RemoveInstanceArgs{
		ClusterID: m.cloudConfig.ClusterID,
		NodeInfos: nodeinfos,
	}
	err = m.service.RemoveInstances(args)
	if err != nil {
		klog.Errorf("failed to remove instance from scaling group %s,because of %s", commonAsg.groupID, err.Error())
		return err
	}
	// prevent from triggering api flow control
	time.Sleep(sdkCoolDownTimeout)

	return nil
}

// GetAsgNodes returns AutoScalingGroup nodes.
func (m *sgcloudManager) GetAsgNodes(sg *AutoScalingGroup) ([]string, error) {
	result := make([]string, 0)
	instances, err := m.service.getScalingInstancesByCluster(sg.manager.cloudConfig.ClusterID)
	if err != nil {
		return []string{}, err
	}
	for _, instance := range instances {
		result = append(result, getNodeProviderID(instance.Id))
	}
	return result, nil
}

// getNodeProviderID build provider id from ecs id and region
func getNodeProviderID(id string) string {
	return fmt.Sprintf("sgcloud://%s", id)
}

func (m *sgcloudManager) getAsgTemplate(asgId string) (*asgTemplate, error) {
	_, err := m.service.getScalingGroupByID(asgId)
	if err != nil {
		klog.V(4).Infof("get scaling group err: %s\n", err)
		return nil, err
	}

	tags := make(map[string]string)
	// for _, tag := range sg.Tags {
	// 	tags[tag.Key] = tag.Value
	// }

	return &asgTemplate{
		InstanceType:     11,
		Region:           m.cloudConfig.Region,
		CPU:              4,
		Memory:           8,
		GpuCount:         0,
		EphemeralStorage: 0,
		Tags:             tags,
	}, nil
}

func (m *sgcloudManager) getAsgTmpl() (*asgTemplate, error) {
	tags := make(map[string]string)
	// for _, tag := range sg.Tags {
	// 	tags[tag.Key] = tag.Value
	// }

	return &asgTemplate{
		InstanceType:     11,
		Region:           m.cloudConfig.Region,
		CPU:              4,
		Memory:           8,
		GpuCount:         0,
		EphemeralStorage: 0,
		Tags:             tags,
	}, nil
}

func (m *sgcloudManager) buildNodeFromTemplate(sg *AutoScalingGroup, template *asgTemplate) (*apiv1.Node, error) {
	node := apiv1.Node{}
	nodeName := fmt.Sprintf("%s-asg-%d", sg.groupID, rand.Int63())

	node.ObjectMeta = metav1.ObjectMeta{
		Name:     nodeName,
		SelfLink: fmt.Sprintf("/api/v1/nodes/%s", nodeName),
		Labels:   map[string]string{},
	}

	node.Status = apiv1.NodeStatus{
		Capacity: apiv1.ResourceList{},
	}

	node.Status.Capacity[apiv1.ResourcePods] = *resource.NewQuantity(defaultPodAmountsLimit, resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceCPU] = *resource.NewQuantity(int64(template.CPU), resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceMemory] = *resource.NewQuantity(int64(template.Memory)*1024*1024*1024, resource.DecimalSI)

	node.Status.Allocatable = node.Status.Capacity

	node.Labels = cloudprovider.JoinStringMaps(node.Labels, buildGenericLabels(template, nodeName))

	node.Status.Conditions = cloudprovider.BuildReadyConditions()
	return &node, nil
}

func buildGenericLabels(template *asgTemplate, nodeName string) map[string]string {
	result := make(map[string]string)
	result[apiv1.LabelArchStable] = cloudprovider.DefaultArch
	result[apiv1.LabelOSStable] = cloudprovider.DefaultOS
	result[apiv1.LabelTopologyRegion] = template.Region
	result[apiv1.LabelTopologyZone] = template.Zone
	result[apiv1.LabelHostname] = nodeName

	// append custom node labels
	for key, value := range template.Tags {
		result[key] = value
	}

	return result
}
