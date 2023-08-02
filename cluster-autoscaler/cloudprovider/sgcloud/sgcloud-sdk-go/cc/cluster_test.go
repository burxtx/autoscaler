package cc

import (
	"fmt"
	"testing"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

const (
	CLUSTERID = ""
	NODE1ID   = ""
	NODE2ID   = ""
)

func initClient() *Client {
	cfg := &core.Config{
		ProxyHost: "127.0.0.1",
		ProxyPort: 11881,
		Endpoint:  "127.0.0.1:11881",
	}
	return NewClient(cfg)
}
func TestDescribeCluster(t *testing.T) {
	c := initClient()
	eg, err := c.DescribeCluster(CLUSTERID)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(eg)
}

func TestListClusterNodes(t *testing.T) {
	c := initClient()
	eg, err := c.ListClusterNodes(CLUSTERID)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(eg)
}

func TestAddInstance(t *testing.T) {
	c := initClient()
	err := c.AddInstance(&AddInstanceArgs{
		ClusterID: CLUSTERID,
		Delta:     2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveInstance(t *testing.T) {
	c := initClient()
	err := c.RemoveInstance(&RemoveInstanceArgs{
		ClusterID: CLUSTERID,
		NodeInfos: []string{
			NODE1ID,
			NODE2ID,
		},
	})
	if err != nil {
		t.Error(err)
	}
}
