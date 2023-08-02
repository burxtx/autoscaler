package cc

import (
	"encoding/json"
	"fmt"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

// ScalingGroup defines autoscaling group
type ScalingGroup struct {
	InstanceType     int    `json:"instanceType"`
	CPU              int    `json:"cpu,omitempty"`
	Memory           int    `json:"memory,omitempty"`
	GpuCount         int    `json:"gpuCount,omitempty"`
	GpuCard          string `json:"gpuCard,omitempty"`
	DiskSize         int    `json:"diskSize,omitempty"`
	EphemeralStorage int    `json:"ephemeralStorage,omitempty"`
	Tags             []Tag  `json:"tags"`
}

// Tag defines label
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"Value"`
}

// DescribeGroup returns the description of the group
func (c *Client) DescribeGroup(groupID string) (*ScalingGroup, error) {
	if groupID == "" {
		return nil, fmt.Errorf("groupID should not be nil")
	}

	params := map[string]string{
		"groupId": groupID,
	}
	req, err := core.NewRequest("GET", c.GetURL("/v1/cluster/group", params), nil)

	if err != nil {
		return nil, err
	}

	resp, err := c.SendRequest(req)

	if err != nil {
		return nil, err
	}

	bodyContent, err := resp.GetBodyContent()

	if err != nil {
		return nil, err
	}

	var sg ScalingGroup
	err = json.Unmarshal(bodyContent, &sg)

	if err != nil {
		return nil, err
	}
	return &sg, nil
}
