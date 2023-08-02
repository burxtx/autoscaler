package sgcloud

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/cc"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

// sgcloudWrapper provides several utility methods over the services provided by the SDK
type sgcloudWrapper struct {
	cc *cc.Client
	// ers *ers.Client
}

func NewsgcloudWrapper(cfg *CloudConfig) (*sgcloudWrapper, error) {
	ccClientCfg := &core.Config{
		Region:   cfg.CcRegion,
		Endpoint: cfg.CcEndpoint,
	}
	cc := cc.NewClient(ccClientCfg)
	return &sgcloudWrapper{
		cc: cc,
	}, nil
}

// getScalingGroupByID describe the scaling group impl required
func (m *sgcloudWrapper) getScalingGroupByID(id string) (*cc.ScalingGroup, error) {
	return m.cc.DescribeGroup(id)
}

// getScalingInstancesByGroup impl required
func (m *sgcloudWrapper) getScalingInstancesByCluster(id string) ([]cc.Instance, error) {
	return m.cc.ListClusterNodes(id)
}

// setCapcityInstanceSize impl required
func (m *sgcloudWrapper) IncreaseCapcityInstanceSize(args *cc.AddInstanceArgs) error {
	return m.cc.AddInstance(args)
}

// RemoveInstances impl required
func (m *sgcloudWrapper) RemoveInstances(args *cc.RemoveInstanceArgs) error {
	return m.cc.RemoveInstance(args)
}

// impl optional
func (m *sgcloudWrapper) getInstanceTags(id string, size int64) error {
	return nil
}
