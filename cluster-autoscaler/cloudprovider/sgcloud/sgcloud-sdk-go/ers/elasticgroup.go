package ers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/core"
)

type ElasticGroupResp struct {
	Data    ElasticGroup `json:"data"`
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Success bool         `json:"success"`
}

type ElasticGroup struct {
	ElasticGroupID string `json:"elasticgroupid"`
	Name           string `json:"name"`
	ResType        int64  `json:"res_type"`
	CCID           string `json:"ccid"` //集群ID
	// TemplateID       string  //启动模板
	ElasticInstances []map[string]interface{} `json:"elasticgroupitems"`
	Notes            string                   `json:"notes"`

	// TeanantID        string
	// ProjectID        string
	// DCRegionID       string
	// DCZoneID         string
	// CreateUser       string
	// CreateUserName   string
	// ModifyUser       string
	// ModifyUserName   string
	// Del              bool
	// CreateDate       time.Time
	// ModifyDate       time.Time
}

// DescribeGroup returns the description of the group
func (c *Client) DescribeElasticGroup(groupID string) (*ElasticGroup, error) {
	if groupID == "" {
		return nil, fmt.Errorf("groupID should not be nil")
	}

	req, err := core.NewRequest(http.MethodGet, c.GetURL(fmt.Sprintf("/api/elasticgroups/%s/get", groupID), nil), nil)

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
	fmt.Println(string(bodyContent))
	var eg ElasticGroupResp
	err = json.Unmarshal(bodyContent, &eg)

	if err != nil {
		return nil, err
	}
	return &eg.Data, nil
}
