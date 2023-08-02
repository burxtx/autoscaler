package cc

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

type Client struct {
	*core.Client
}

// Endpoint contains all endpoints of sgcloud.
var Endpoint = map[string]string{
	"bqj":   "sgcloud_ers_service",
	"debug": "",
}

// NewClient returns client for BCC
func NewClient(config *core.Config) *Client {
	ccClient := core.NewClient(config)
	return &Client{ccClient}
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
