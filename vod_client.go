package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
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

// VODClient VOD API客户端（自定义实现，已废弃，建议使用VODSDKClient）
type VODClient struct {
	accessKey  string
	secretKey  string
	region     string
	apiEndpoint string
	httpClient *http.Client
}

// NewVODClient 创建VOD客户端
func NewVODClient(accessKey, secretKey, region, apiEndpoint string) *VODClient {
	return &VODClient{
		accessKey:   accessKey,
		secretKey:   secretKey,
		region:      region,
		apiEndpoint: apiEndpoint,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// APIResponse API响应结构
type APIResponse struct {
	ResponseMetadata struct {
		RequestID string `json:"RequestId"`
		Action    string `json:"Action"`
		Version   string `json:"Version"`
		Service   string `json:"Service"`
		Region    string `json:"Region"`
	} `json:"ResponseMetadata"`
	Result interface{} `json:"Result"`
}

// StartWorkflowResponse 启动工作流响应
type StartWorkflowResponse struct {
	RunId string `json:"RunId"`
}

// GetWorkflowExecutionResponse 获取工作流执行状态响应
type GetWorkflowExecutionResponse struct {
	RunId    string `json:"RunId"`
	Vid      string `json:"Vid"`
	Status   string `json:"Status"`
	TemplateId string `json:"TemplateId"`
	TemplateName string `json:"TemplateName"`
	SpaceName string `json:"SpaceName"`
	CreateTime string `json:"CreateTime"`
	StartTime  string `json:"StartTime"`
	EndTime    string `json:"EndTime"`
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

// signRequest 对请求进行签名
// 注意：这是简化的签名实现，实际使用时建议使用火山引擎官方SDK
// 火山引擎API签名规范：https://www.volcengine.com/docs/4/3479
func (c *VODClient) signRequest(method, uri string, params url.Values) string {
	// 1. 规范化参数（按字典序排序）
	keys := make([]string, 0, len(params))
	for k := range params {
		// 排除Signature参数本身
		if k != "Signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 2. 构建规范化查询字符串
	var canonicalQueryString strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonicalQueryString.WriteString("&")
		}
		// URL编码参数名和参数值
		canonicalQueryString.WriteString(url.QueryEscape(k))
		canonicalQueryString.WriteString("=")
		canonicalQueryString.WriteString(url.QueryEscape(params.Get(k)))
	}

	// 3. 构建待签名字符串
	// 格式: Method + "\n" + URI + "\n" + CanonicalQueryString
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s", method, uri, canonicalQueryString.String())

	// 4. 使用HMAC-SHA256计算签名
	// 火山引擎的Secret Key是base64编码的，需要先解码
	secretKeyBytes, err := base64.StdEncoding.DecodeString(c.secretKey)
	if err != nil {
		// 如果解码失败，尝试直接使用原始字符串
		secretKeyBytes = []byte(c.secretKey)
	}
	
	mac := hmac.New(sha256.New, secretKeyBytes)
	mac.Write([]byte(canonicalRequest))
	signatureBytes := mac.Sum(nil)
	
	// 火山引擎API要求签名使用hex编码（小写）
	signature := strings.ToLower(hex.EncodeToString(signatureBytes))

	return signature
}

// callAPI 调用VOD API
func (c *VODClient) callAPI(action string, params url.Values) ([]byte, error) {
	// 添加公共参数
	params.Set("Action", action)
	params.Set("Version", "2020-08-01")
	params.Set("AccessKeyId", c.accessKey)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	params.Set("Timestamp", timestamp)
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureMethod", "HMAC-SHA256")

	// 签名（注意：需要在添加Signature之前计算）
	uri := "/"
	signature := c.signRequest("GET", uri, params)
	params.Set("Signature", signature)

	// 构建请求URL
	reqURL := fmt.Sprintf("%s?%s", c.apiEndpoint, params.Encode())

	// 发送HTTP GET请求
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp struct {
			ResponseMetadata struct {
				RequestID string `json:"RequestId"`
				Error     struct {
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"Error"`
			} `json:"ResponseMetadata"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorResp.ResponseMetadata.Error.Code != "" {
				return nil, fmt.Errorf("API返回错误: Code=%s, Message=%s, RequestId=%s", 
					errorResp.ResponseMetadata.Error.Code, 
					errorResp.ResponseMetadata.Error.Message,
					errorResp.ResponseMetadata.RequestID)
			}
		}
		return nil, fmt.Errorf("API调用失败: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 检查响应中是否包含错误信息
	var errorResp struct {
		ResponseMetadata struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"ResponseMetadata"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorResp.ResponseMetadata.Error.Code != "" {
			return nil, fmt.Errorf("API返回错误: Code=%s, Message=%s", 
				errorResp.ResponseMetadata.Error.Code, 
				errorResp.ResponseMetadata.Error.Message)
		}
	}

	return body, nil
}

// StartWorkflow 启动工作流
func (c *VODClient) StartWorkflow(spaceName, workflowId, fileName, bucketName string) (string, error) {
	params := url.Values{}
	params.Set("SpaceName", spaceName)
	params.Set("TemplateId", workflowId)
	
	// Input参数需要是JSON格式
	inputJSON := fmt.Sprintf(`{"FileName":"%s","BucketName":"%s"}`, fileName, bucketName)
	params.Set("Input", inputJSON)

	body, err := c.callAPI("StartWorkflow", params)
	if err != nil {
		return "", err
	}

	var response struct {
		ResponseMetadata struct {
			RequestID string `json:"RequestId"`
		} `json:"ResponseMetadata"`
		Result struct {
			RunId string `json:"RunId"`
		} `json:"Result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	if response.Result.RunId == "" {
		return "", fmt.Errorf("未获取到RunId, 响应: %s", string(body))
	}

	return response.Result.RunId, nil
}

// GetWorkflowExecution 获取工作流执行状态
func (c *VODClient) GetWorkflowExecution(runId string) (*GetWorkflowExecutionResponse, error) {
	params := url.Values{}
	params.Set("RunId", runId)

	body, err := c.callAPI("GetWorkflowExecution", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		ResponseMetadata struct {
			RequestID string `json:"RequestId"`
		} `json:"ResponseMetadata"`
		Result GetWorkflowExecutionResponse `json:"Result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	return &response.Result, nil
}

// GetPlayInfo 获取播放地址
func (c *VODClient) GetPlayInfo(vid, format, definition string) (*GetPlayInfoResponse, error) {
	params := url.Values{}
	params.Set("Vid", vid)
	params.Set("Format", format)
	params.Set("Definition", definition)
	params.Set("FileType", "video")

	body, err := c.callAPI("GetPlayInfo", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		ResponseMetadata struct {
			RequestID string `json:"RequestId"`
		} `json:"ResponseMetadata"`
		Result GetPlayInfoResponse `json:"Result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	return &response.Result, nil
}

