package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/vod"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/business"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

// VODClientInterface VOD客户端接口
type VODClientInterface interface {
	StartWorkflow(spaceName, workflowId, fileName, bucketName, clientToken string) (string, error)
	GetWorkflowExecution(runId string) (*GetWorkflowExecutionResponse, error)
	GetWorkflowExecutionResult(runId string) (string, *TranscodeInfo, error)
	GetPlayInfo(vid, format, definition string) (*GetPlayInfoResponse, error)
	GetPlayInfoByFilePath(filePath, format, definition string, transcodeInfo *TranscodeInfo) (*GetPlayInfoResponse, error)
	GetMediaInfos(filePath string) (string, error)
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
	Vid          string  `json:"Vid"`
	Status       int     `json:"Status"`
	PosterUrl    string  `json:"PosterUrl"`
	Duration     float64 `json:"Duration"`
	FileType     string  `json:"FileType"`
	TotalCount   int     `json:"TotalCount"`
	PlayInfoList []struct {
		FileId        string `json:"FileId"`
		FileType      string `json:"FileType"`
		Definition    string `json:"Definition"`
		Format        string `json:"Format"`
		Codec         string `json:"Codec"`
		MainPlayUrl   string `json:"MainPlayUrl"`
		BackupPlayUrl string `json:"BackupPlayUrl"`
		Bitrate       int    `json:"Bitrate"`
		Width         int    `json:"Width"`
		Height        int    `json:"Height"`
		Size          int64  `json:"Size"`
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
func (c *VODClient) StartWorkflow(spaceName, workflowId, fileName, bucketName, clientToken string) (string, error) {
	// 根据文档，StartWorkflow需要使用DirectUrl方式指定TOS中的文件
	directUrl := &business.DirectUrl{
		FileName:   fileName,
		BucketName: bucketName,
		SpaceName:  spaceName,
	}

	input := &request.VodStartWorkflowRequest{
		TemplateId:  workflowId,
		DirectUrl:   directUrl,
		ClientToken: strings.TrimSpace(clientToken),
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

// TranscodeInfo 转码信息结构体
type TranscodeInfo struct {
	StoreUri string  // 转码后的文件路径
	Format   string  // 文件格式
	Duration float32 // 时长
	Size     float64 // 文件大小
	Width    int     // 宽度
	Height   int     // 高度
	Bitrate  int     // 码率
}

// GetWorkflowExecutionResult 获取工作流执行结果（包含转码产物信息）
func (c *VODClient) GetWorkflowExecutionResult(runId string) (string, *TranscodeInfo, error) {
	// 调用GetWorkflowExecutionResult API获取转码结果
	input := &request.VodGetWorkflowResultRequest{
		RunId: runId,
	}

	output, _, err := c.client.GetWorkflowExecutionResult(input)
	if err != nil {
		return "", nil, fmt.Errorf("调用GetWorkflowExecutionResult失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return "", nil, fmt.Errorf("API返回错误: Code=%s, Message=%s",
			output.ResponseMetadata.Error.Code,
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil {
		return "", nil, fmt.Errorf("未获取到结果")
	}

	// 从结果中获取Vid
	vid := output.Result.Vid

	var transcodeInfo *TranscodeInfo
	if output.Result.TranscodeInfos != nil && len(output.Result.TranscodeInfos) > 0 {
		bestScore := -1
		var bestStoreUri string
		var bestFormat string
		var bestDuration float32
		var bestSize float64
		var bestWidth int
		var bestHeight int
		var bestBitrate int

		for _, info := range output.Result.TranscodeInfos {
			storeUri := strings.TrimSpace(info.StoreUri)
			if storeUri == "" {
				continue
			}

			score := 0
			if strings.EqualFold(info.FileType, "video") {
				score += 10
			}
			if strings.EqualFold(info.Format, "mp4") {
				score += 5
			}
			if info.VideoStreamMeta != nil {
				score += 1
			}

			if score > bestScore {
				bestScore = score
				bestStoreUri = storeUri
				bestFormat = info.Format
				bestDuration = info.Duration
				bestSize = info.Size
				if info.VideoStreamMeta != nil {
					bestWidth = int(info.VideoStreamMeta.Width)
					bestHeight = int(info.VideoStreamMeta.Height)
					bestBitrate = int(info.VideoStreamMeta.Bitrate)
				} else {
					bestWidth = 0
					bestHeight = 0
					bestBitrate = 0
				}
			}
		}

		if bestScore >= 0 {
			transcodeInfo = &TranscodeInfo{
				StoreUri: bestStoreUri,
				Format:   bestFormat,
				Duration: bestDuration,
				Size:     bestSize,
				Width:    bestWidth,
				Height:   bestHeight,
				Bitrate:  bestBitrate,
			}
		}
	}

	return vid, transcodeInfo, nil
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
			FileId        string `json:"FileId"`
			FileType      string `json:"FileType"`
			Definition    string `json:"Definition"`
			Format        string `json:"Format"`
			Codec         string `json:"Codec"`
			MainPlayUrl   string `json:"MainPlayUrl"`
			BackupPlayUrl string `json:"BackupPlayUrl"`
			Bitrate       int    `json:"Bitrate"`
			Width         int    `json:"Width"`
			Height        int    `json:"Height"`
			Size          int64  `json:"Size"`
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

// GetMediaInfos 通过Vid获取媒体信息
func (c *VODClient) GetMediaInfos(vid string) (string, error) {
	// 使用GetMediaInfos API通过Vid获取媒体信息
	input := &request.VodGetMediaInfosRequest{
		Vids: vid,
	}

	output, _, err := c.client.GetMediaInfos(input)
	if err != nil {
		return "", fmt.Errorf("调用GetMediaInfos失败: %v", err)
	}

	if output.ResponseMetadata != nil && output.ResponseMetadata.Error != nil {
		return "", fmt.Errorf("API返回错误: Code=%s, Message=%s",
			output.ResponseMetadata.Error.Code,
			output.ResponseMetadata.Error.Message)
	}

	if output.Result == nil {
		return "", fmt.Errorf("未找到媒体信息")
	}

	return vid, nil
}

// GetPlayInfoByFilePath 通过文件路径获取播放地址（DirectUrl模式）
// 在DirectUrl模式下，我们需要从工作流执行结果中获取转码后的文件信息
// 这里我们通过检查工作流执行状态时保存的信息来获取转码结果
func (c *VODClient) GetPlayInfoByFilePath(filePath, format, definition string, transcodeInfo *TranscodeInfo) (*GetPlayInfoResponse, error) {
	// 获取TOS桶的访问域名
	// 在实际项目中，应该从配置中获取正确的TOS访问域名
	tosDomain := "dji-vod-poc.volces.com"

	var playUrl string
	var width, height, bitrate int
	var size int64
	var duration float64

	if transcodeInfo != nil && transcodeInfo.StoreUri != "" {
		// 使用从工作流执行结果中获取的转码信息生成播放地址
		fmt.Printf("使用工作流转码结果生成播放地址: StoreUri=%s\n", transcodeInfo.StoreUri)
		playUrl = fmt.Sprintf("https://%s/%s", tosDomain, transcodeInfo.StoreUri)
		width = transcodeInfo.Width
		height = transcodeInfo.Height
		bitrate = transcodeInfo.Bitrate
		size = int64(transcodeInfo.Size)
		duration = float64(transcodeInfo.Duration)
	} else {
		// 如果没有提供转码信息，使用备用方式生成转码后的文件路径
		fmt.Println("未获取到转码信息，使用备用方式生成播放地址")
		// 方式1：在原文件名后添加_720p后缀
		transcodedFilePath := filePath[:len(filePath)-len(".mp4")] + "_720p.mp4"
		playUrl = fmt.Sprintf("https://%s/%s", tosDomain, transcodedFilePath)
		// 使用默认的清晰度参数
		width = 1280
		height = 720
		bitrate = 2000000
	}

	// 注意：如果TOS桶没有设置公开访问，需要生成带有签名的临时URL
	// 可以使用TOS SDK的PresignedGetObject方法生成临时URL

	// 返回包含真实播放地址的GetPlayInfoResponse结构
	return &GetPlayInfoResponse{
		Vid:        "",
		Status:     10, // 10表示成功
		PosterUrl:  "",
		Duration:   duration,
		FileType:   "video",
		TotalCount: 1,
		PlayInfoList: []struct {
			FileId        string `json:"FileId"`
			FileType      string `json:"FileType"`
			Definition    string `json:"Definition"`
			Format        string `json:"Format"`
			Codec         string `json:"Codec"`
			MainPlayUrl   string `json:"MainPlayUrl"`
			BackupPlayUrl string `json:"BackupPlayUrl"`
			Bitrate       int    `json:"Bitrate"`
			Width         int    `json:"Width"`
			Height        int    `json:"Height"`
			Size          int64  `json:"Size"`
		}{
			{
				FileId:        "",
				FileType:      "video",
				Definition:    definition,
				Format:        format,
				Codec:         "H264",
				MainPlayUrl:   playUrl,
				BackupPlayUrl: "",
				Bitrate:       bitrate,
				Width:         width,
				Height:        height,
				Size:          size,
			},
		},
	}, nil
}
