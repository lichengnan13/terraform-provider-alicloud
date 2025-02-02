package alicloud

import (
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ons"
	"github.com/terraform-providers/terraform-provider-alicloud/alicloud/connectivity"
)

type OnsService struct {
	client *connectivity.AliyunClient
}

func (s *OnsService) InstanceNotExistFunc(err error) bool {
	return strings.Contains(err.Error(), OnsInstanceNotExist)
}

func (s *OnsService) GetPreventCache() requests.Integer {
	return requests.NewInteger(int(time.Now().UnixNano() / 1e6))
}

func (s *OnsService) DescribeOnsInstance(id string) (*ons.OnsInstanceBaseInfoResponse, error) {
	request := ons.CreateOnsInstanceBaseInfoRequest()
	request.PreventCache = s.GetPreventCache()
	request.InstanceId = id

	raw, err := s.client.WithOnsClient(func(onsClient *ons.Client) (interface{}, error) {
		return onsClient.OnsInstanceBaseInfo(request)
	})

	if err != nil {
		if IsExceptedError(err, InvalidDomainNameNoExist) {
			return nil, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return nil, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
	}

	response, _ := raw.(*ons.OnsInstanceBaseInfoResponse)
	addDebug(request.GetActionName(), raw)
	return response, nil

}

func (s *OnsService) DescribeOnsTopic(id string) (onsTopic *ons.PublishInfoDo, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return nil, WrapError(err)
	}
	instanceId := parts[0]
	topic := parts[1]

	request := ons.CreateOnsTopicListRequest()
	request.InstanceId = instanceId
	request.PreventCache = s.GetPreventCache()

	raw, err := s.client.WithOnsClient(func(onsClient *ons.Client) (interface{}, error) {
		return onsClient.OnsTopicList(request)
	})

	if err != nil {
		if IsExceptedErrors(err, []string{AuthResourceOwnerError, OnsInstanceNotExist}) {
			return onsTopic, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return onsTopic, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
	}

	topicListResp, _ := raw.(*ons.OnsTopicListResponse)
	addDebug(request.GetActionName(), raw)

	for _, v := range topicListResp.Data.PublishInfoDo {
		if v.Topic == topic {
			onsTopic = &v
			return
		}
	}
	return onsTopic, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
}

func (s *OnsService) DescribeOnsGroup(id string) (onsGroup *ons.SubscribeInfoDo, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return nil, WrapError(err)
	}
	instanceId := parts[0]
	groupId := parts[1]

	request := ons.CreateOnsGroupListRequest()
	request.InstanceId = instanceId
	request.PreventCache = s.GetPreventCache()

	raw, err := s.client.WithOnsClient(func(onsClient *ons.Client) (interface{}, error) {
		return onsClient.OnsGroupList(request)
	})
	if err != nil {
		if IsExceptedErrors(err, []string{AuthResourceOwnerError, OnsInstanceNotExist}) {
			return onsGroup, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return onsGroup, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
	}

	groupListResp, _ := raw.(*ons.OnsGroupListResponse)
	addDebug(request.GetActionName(), raw)

	for _, v := range groupListResp.Data.SubscribeInfoDo {
		if v.GroupId == groupId {
			onsGroup = &v
			return
		}
	}

	return onsGroup, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
}

func (s *OnsService) WaitForOnsInstance(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		response, err := s.DescribeOnsInstance(id)

		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if response.InstanceBaseInfo.InstanceId == id && status != Deleted {
			return nil
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, response.InstanceBaseInfo.InstanceId, id, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}

}

func (s *OnsService) WaitForOnsTopic(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return WrapError(err)
	}
	instanceId := parts[0]
	topic := parts[1]
	for {
		response, err := s.DescribeOnsTopic(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if response.InstanceId+":"+response.Topic == id && status != Deleted {
			return nil
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, instanceId+":"+topic, id, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *OnsService) WaitForOnsGroup(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		response, err := s.DescribeOnsGroup(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if response.InstanceId+":"+response.GroupId == id && status != Deleted {
			return nil
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, response.InstanceId+":"+response.GroupId, id, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}
