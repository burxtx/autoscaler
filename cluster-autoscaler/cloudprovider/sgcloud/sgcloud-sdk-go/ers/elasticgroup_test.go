package ers

import (
	"fmt"
	"testing"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

func initClient() *Client {
	cfg := &core.Config{
		ProxyHost: "127.0.0.1",
		ProxyPort: 11881,
		Endpoint:  "127.0.0.1:11881",
		Debug:     true,
	}
	return NewClient(cfg)
}
func TestDescribeGroup(t *testing.T) {
	c := initClient()
	eg, err := c.DescribeElasticGroup("1678324274848337920")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(eg)
}
