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
)

type CloudConfig struct {
	ClusterID       string `json:"ClusterId"`
	ClusterName     string `json:"ClusterName"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Region          string `json:"Region"`
	CcRegion        string `json:"CcRegion"`
	ErsRegion       string `json:"ErsRegion"`
	VpcID           string `json:"VpcId"`
	MasterID        string `json:"MasterId"`
	CcEndpoint      string `json:"CcEndpoint"`
	ErsEndpoint     string `json:"ErsEndpoint"`
	NodeIP          string `json:"NodeIP"`
	Debug           bool   `json:"Debug"`
}

func (cc *CloudConfig) validate() error {
	// if cc.MasterID == "" {
	// 	return fmt.Errorf("sgcloud: Cloud config must have a Master ID")
	// }
	if cc.ClusterID == "" {
		return fmt.Errorf("sgcloud: Cloud config must have a ClusterID")
	}
	// if cc.Endpoint == "" {
	// 	return fmt.Errorf("sgcloud: Cloud config must have a Endpoint")
	// }
	return nil
}
