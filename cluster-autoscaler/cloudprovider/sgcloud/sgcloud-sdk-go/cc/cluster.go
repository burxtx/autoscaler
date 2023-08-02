package cc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

type NodeConfig struct {
	InstanceType int    `json:"instanceType"`
	CPU          int    `json:"cpu,omitempty"`
	Memory       int    `json:"memory,omitempty"`
	GpuCount     int    `json:"gpuCount,omitempty"`
	GpuCard      string `json:"gpuCard,omitempty"`
	DiskSize     int    `json:"diskSize,omitempty"`
}

type InstanceResp struct {
	Data    Data   `json:"data"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}
type Data struct {
	PageItems  []Instance `json:"pageItems"`
	PageNumber int        `json:"pageNumber"`
	PageCount  string     `json:"pageCount"`
}
type Instance struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Role   int    `json:"role"`
	Status int    `json:"status"`
	Spec   string `json:"spec"`
	CpuUse string `json:"cpuUse"`
	MemUse string `json:"memUse"`
	UpTime string `json:"upTime"`
	VmId   string `json:"vmId"`
	NodeId string `json:"nodeId"`
}

type AddInstanceArgs struct {
	ClusterID string `json:"clusterId"`
	Delta     int    `json:"delta"`
}

type RemoveInstanceArgs struct {
	ClusterID string   `json:"clusterId"`
	NodeInfos []string `json:"nodeId"`
}

type ContainerCluster struct {
	ID                string
	Name              string
	ArchType          int32
	GuestOS           int32
	ClusterType       int32
	K8sVersion        int32
	ContainerRuntime  int32
	NetworkPluginType int32
	PodNetworkCider   string
	ServiceCider      string
	SnatToggle        int32
	EipToggle         int32
	VpcNetowrk        string
	VolumeType        int32
	StoragePluginType int32
	Dss               int32
	SecretCode        string
	SecretCodeConfirm string
	SecurityGroupType int32
	NodeCount         int32
	ClusterName       string

	// DcRegionID        int64
	// DcZoneID          int64
	// DelFlag           int32
	// ProjectID         int64
	// TeanantID         int64
	// CreateUser        string
	// CreateUserName    string
	// CreateDate        time.Time
	// ModifyUser        string
	// ModifyUserName    string
	// ModifyDate        time.Time
}

// DescribeCluster returns the description of the cluster
func (c *Client) DescribeCluster(clusterID string) (*ContainerCluster, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("clusterID should not be nil")
	}

	req, err := core.NewRequest(http.MethodPost, c.GetURL("/cluster/get", nil), map[string]string{"id": clusterID})

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

	var cc ContainerCluster
	err = json.Unmarshal(bodyContent, &cc)

	if err != nil {
		return nil, err
	}
	return &cc, nil
}

// RemoveInstance returns the description of the cluster
func (c *Client) RemoveInstance(args *RemoveInstanceArgs) error {
	postContent, err := json.Marshal(args)
	if err != nil {
		return err
	}

	req, err := core.NewRequest(http.MethodPost, c.GetURL("/cluster/nodes/delete", nil), bytes.NewBuffer(postContent))

	if err != nil {
		return err
	}

	_, err = c.SendRequest(req)
	return err
}

// AddInstance returns the description of the cluster
func (c *Client) AddInstance(args *AddInstanceArgs) error {
	postContent, err := json.Marshal(args)
	if err != nil {
		return err
	}

	req, err := core.NewRequest(http.MethodPost, c.GetURL("/cluster/nodes/add", nil), bytes.NewBuffer(postContent))

	if err != nil {
		return err
	}

	_, err = c.SendRequest(req)
	return err
}

// DescribeCluster returns the description of the cluster
func (c *Client) ListClusterNodes(clusterID string) ([]Instance, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("clusterID should not be nil")
	}

	req, err := core.NewRequest(http.MethodPost, c.GetURL("/cluster/nodes", nil), map[string]interface{}{
		"filter": map[string]string{
			"clusterId": clusterID,
		},
		"pageIndex": 1,
		"pageSize":  1000,
		"sorter":    nil,
	})

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

	var ins InstanceResp
	err = json.Unmarshal(bodyContent, &ins)

	if err != nil {
		return nil, err
	}
	return ins.Data.PageItems, nil
}
