package client

import (
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hongfs/ecs-metadata/pkg/metadata"
	"os"
)

func ECS(ram string) (*ecs20140526.Client, error) {
	info := metadata.Ram(ram)

	if info == nil {
		return nil, metadata.ErrRamInfoNil
	}

	region := metadata.Region()

	endpoint := fmt.Sprintf("ecs-vpc.%s.aliyuncs.com", region)

	return ecs20140526.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(info.AccessKeyID),
		AccessKeySecret: tea.String(info.AccessKeySecret),
		SecurityToken:   tea.String(info.SecurityToken),
		RegionId:        tea.String(region),
		Endpoint:        tea.String(endpoint),
	})
}

func EcsForDefault() (*ecs20140526.Client, error) {
	return ECS(os.Getenv("ECS_RAM_NAME"))
}
