package main

import (
	"fmt"
	"time"

	"github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/vod"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/business"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

// VODClientInterface VOD客户端接口
type VODClientInterface interface {
	StartWorkflow(spaceName, workflowId, fileName, bucketName string) (string, error)
	GetWorkflowExecution(runId string) (*GetWorkflowExecutionResponse, error)
	GetPlayInfo(vid, format, definition string) (*GetPlayInfoResponse, error)
}

// GetWorkflowExecutionResultInterface 获取工作流执行结果接口（可选）
type GetWorkflowExecutionResultInterface interface {
	GetWorkflowExecutionResult(runId string) (string, error)
}

// GetWorkflowExecutionResponse 获取工作流执行状态响应
type GetWorkflowExecutionResponse struct {
	RunId        string `json:"RunId"`
	Vid          string `json:"Vid"`
	Status       string `json:"Status"`
	TemplateId   string `json:"TemplateId"`
	TemplateName string `json:"TemplateName"`
	SpaceName    string `json:"SpaceName"`
	CreateTime   string `json:"CreateTime"`
	StartTime    string `json:"StartTime"`
	EndTime      string `json:"EndTime"`
}

// GetPlayInfoResponse 获取播放地址响应
type GetPlayInfoResponse struct {
	Vid         string `json:"Vid"`
	Status      int    `json:"Status"`
	PosterUrl   string `json:"PosterUrl"`
	Duration    float64 `json:"Duration"`
	FileType    string `json:"FileType"`
	TotalCount  int    `json:"TotalCount"`
	PlayInfoList []struct {
		FileId         string `json:"FileId"`
		FileType       string `json:"FileType"`
		Definition     string `json:"Definition"`
		Format         string `json:"Format"`
		Codec          string `json:"Codec"`
		MainPlayUrl    string `json:"MainPlayUrl"`
		BackupPlayUrl  string `json:"BackupPlayUrl"`
		Bitrate        int    `json:"Bitrate"`
		Width          int    `json:"Width"`
		Height         int    `json:"Height"`
		Size           int64  `json:"Size"`
	} `json:"PlayInfoList"`
}

// VODClient 使用volc-sdk-golang V1.0的VOD客户端
// 注意：当前使用V1.0 SDK，因为V2.0 SDK (volcengine-go-sdk) 目前不支持StartWorkflow API
// V2.0 SDK只支持StartExecution API，而我们需要使用StartWorkflow来触发转码工作流
// 参考文档：docs/V2.0 Go SDK 介绍与迁移说明.md
type VODClient struct {
	client *vod.Vod
	region string
}

// NewVODClient 创建VOD客户端
// 使用V1.0 SDK (github.com/volcengine/volc-sdk-golang/service/vod)
func NewVODClient(accessKey, secretKey, region, apiEndpoint string) VODClientInterface {
	instance := vod.NewInstanceWithRegion(region)
	instance.SetCredential(base.Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	return &VODClient{
		client: instance,
		region: region,
	}
}

// StartWorkflow 启动工作流
func (c *VODClient) StartWorkflow(spaceName, workflowId, fileName, bucketName string) (string, error) {
	// 根据文档，StartWorkflow需要使用DirectUrl方式指定TOS中的文件
	directUrl := &business.DirectUrl{
		FileName:   fileName,
		BucketName: bucketName,
		SpaceName:  spaceName,
	}
	
	input := &request.VodStartWorkflowRequest{
		TemplateId: workflowId,
		DirectUrl:  directUrl,
	}

	output, _, err := c.client.StartWorkflow(input)
	if err != nil {
		return "", fmt.Errorf("调用StartWorkflow失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return "", fmt.Errorf("API返回错误: Code=%s, Message=%s", 
			output.ResponseMetadata.Error.Code, 
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil || output.Result.RunId == "" {
		return "", fmt.Errorf("未获取到RunId")
	}

	return output.Result.RunId, nil
}

// GetWorkflowExecution 获取工作流执行状态
func (c *VODClient) GetWorkflowExecution(runId string) (*GetWorkflowExecutionResponse, error) {
	input := &request.VodGetWorkflowExecutionStatusRequest{
		RunId: runId,
	}

	output, _, err := c.client.GetWorkflowExecution(input)
	if err != nil {
		return nil, fmt.Errorf("调用GetWorkflowExecution失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return nil, fmt.Errorf("API返回错误: Code=%s, Message=%s", 
			output.ResponseMetadata.Error.Code, 
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil {
		return nil, fmt.Errorf("未获取到结果")
	}

	result := &GetWorkflowExecutionResponse{
		RunId:        output.Result.RunId,
		Status:       output.Result.Status,
		TemplateId:   output.Result.TemplateId,
		TemplateName: output.Result.TemplateName,
		SpaceName:    output.Result.SpaceName,
	}

	// 获取Vid
	if output.Result.Vid != "" {
		result.Vid = output.Result.Vid
	}

	// 转换时间戳
	if output.Result.CreateTime != nil {
		result.CreateTime = output.Result.CreateTime.AsTime().Format(time.RFC3339)
	}
	if output.Result.StartTime != nil {
		result.StartTime = output.Result.StartTime.AsTime().Format(time.RFC3339)
	}
	if output.Result.EndTime != nil {
		result.EndTime = output.Result.EndTime.AsTime().Format(time.RFC3339)
	}

	return result, nil
}

// GetWorkflowExecutionResult 获取工作流执行结果（包含转码产物信息）
func (c *VODClient) GetWorkflowExecutionResult(runId string) (string, error) {
	// 调用GetWorkflowExecutionResult API获取转码结果
	input := &request.VodGetWorkflowResultRequest{
		RunId: runId,
	}

	output, _, err := c.client.GetWorkflowExecutionResult(input)
	if err != nil {
		return "", fmt.Errorf("调用GetWorkflowExecutionResult失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return "", fmt.Errorf("API返回错误: Code=%s, Message=%s", 
			output.ResponseMetadata.Error.Code, 
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil {
		return "", fmt.Errorf("未获取到结果")
	}

	// 从结果中获取Vid
	if output.Result.Vid != "" {
		return output.Result.Vid, nil
	}

	return "", fmt.Errorf("未找到Vid")
}

// GetPlayInfo 获取播放地址
func (c *VODClient) GetPlayInfo(vid, format, definition string) (*GetPlayInfoResponse, error) {
	input := &request.VodGetPlayInfoRequest{
		Vid:        vid,
		Format:     format,
		Definition: definition,
		FileType:   "video",
	}

	output, _, err := c.client.GetPlayInfo(input)
	if err != nil {
		return nil, fmt.Errorf("调用GetPlayInfo失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return nil, fmt.Errorf("API返回错误: Code=%s, Message=%s", 
			output.ResponseMetadata.Error.Code, 
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil {
		return nil, fmt.Errorf("未获取到结果")
	}

	result := &GetPlayInfoResponse{
		Vid:        output.Result.Vid,
		Status:     int(output.Result.Status),
		PosterUrl:  output.Result.PosterUrl,
		Duration:   float64(output.Result.Duration),
		FileType:   output.Result.FileType,
		TotalCount: int(output.Result.TotalCount),
	}

	// 转换PlayInfoList
	if output.Result.PlayInfoList != nil {
		result.PlayInfoList = make([]struct {
			FileId         string `json:"FileId"`
			FileType       string `json:"FileType"`
			Definition     string `json:"Definition"`
			Format         string `json:"Format"`
			Codec          string `json:"Codec"`
			MainPlayUrl    string `json:"MainPlayUrl"`
			BackupPlayUrl  string `json:"BackupPlayUrl"`
			Bitrate        int    `json:"Bitrate"`
			Width          int    `json:"Width"`
			Height         int    `json:"Height"`
			Size           int64  `json:"Size"`
		}, len(output.Result.PlayInfoList))

		for i, playInfo := range output.Result.PlayInfoList {
			result.PlayInfoList[i].FileId = playInfo.FileId
			result.PlayInfoList[i].FileType = playInfo.FileType
			result.PlayInfoList[i].Definition = playInfo.Definition
			result.PlayInfoList[i].Format = playInfo.Format
			result.PlayInfoList[i].Codec = playInfo.Codec
			result.PlayInfoList[i].MainPlayUrl = playInfo.MainPlayUrl
			result.PlayInfoList[i].BackupPlayUrl = playInfo.BackupPlayUrl
			result.PlayInfoList[i].Bitrate = int(playInfo.Bitrate)
			result.PlayInfoList[i].Width = int(playInfo.Width)
			result.PlayInfoList[i].Height = int(playInfo.Height)
			result.PlayInfoList[i].Size = int64(playInfo.Size)
		}
	}

	return result, nil
}
