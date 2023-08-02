package ers

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

type Client struct {
	*core.Client
}

// Endpoint contains all endpoints of Baidu Cloud BCC.
var Endpoint = map[string]string{
	"bqj": "sgcloud_ers_service",
}

// NewClient returns client for BCC
func NewClient(config *core.Config) *Client {
	ersClient := core.NewClient(config)
	return &Client{ersClient}
}

// GetURL generates the full URL of http request.
func (c *Client) GetURL(objectKey string, params map[string]string) string {
	host := c.Endpoint
	if host == "" {
		host = Endpoint[c.GetRegion()]
	}
	uriPath := objectKey
	return c.Client.GetURL(host, uriPath, params)
}
