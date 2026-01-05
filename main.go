package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	TOS   TOSConfig   `yaml:"tos"`
	VOD   VODConfig   `yaml:"vod"`
	Video VideoConfig `yaml:"video"`
}

type TOSConfig struct {
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	Endpoint   string `yaml:"endpoint"`
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
}

type VODConfig struct {
	AccessKey   string `yaml:"access_key"`
	SecretKey   string `yaml:"secret_key"`
	Region      string `yaml:"region"`
	SpaceName   string `yaml:"space_name"`
	WorkflowID  string `yaml:"workflow_id"`
	APIEndpoint string `yaml:"api_endpoint"`
}

type VideoConfig struct {
	InputPath       string `yaml:"input_path"`
	OutputKeyPrefix string `yaml:"output_key_prefix"`
}

// loadEnvFile 加载.env文件
func loadEnvFile(envPath string) error {
	file, err := os.Open(envPath)
	if err != nil {
		// .env文件不存在时不报错，使用环境变量
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 解析 KEY=VALUE 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 移除引号（如果有）
			value = strings.Trim(value, `"'`)
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

// 加载配置
func loadConfig(configPath string) (*Config, error) {
	// 先加载.env文件
	loadEnvFile(".env")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 从环境变量获取密钥（优先级：环境变量 > 配置文件）
	accessKey := os.Getenv("ACCESS_KEY")
	secretKey := os.Getenv("SECRET_KEY")

	if accessKey != "" {
		config.TOS.AccessKey = accessKey
		config.VOD.AccessKey = accessKey
	}
	if secretKey != "" {
		config.TOS.SecretKey = secretKey
		config.VOD.SecretKey = secretKey
	}
	if envVal := os.Getenv("VOD_WORKFLOW_ID"); envVal != "" {
		config.VOD.WorkflowID = envVal
	}

	return &config, nil
}

func newTOSClient(cfg TOSConfig) (*tos.ClientV2, error) {
	client, err := tos.NewClientV2(
		cfg.Endpoint,
		tos.WithRegion(cfg.Region),
		tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey)),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseLastModified(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("LastModified为空")
	}

	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("无法解析LastModified: %s", value)
}

func buildTranscodedOutputKey(inputObjectKey string) string {
	dir := path.Dir(inputObjectKey)
	base := path.Base(inputObjectKey)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		stem = base
	}

	outName := stem + "-转码后-720P.mp4"
	if dir == "." || dir == "/" {
		return outName
	}
	return path.Join(dir, outName)
}

func findTranscodedSourceObjectKey(ctx context.Context, client *tos.ClientV2, bucket, prefix, inputKey, outputKey string, notBefore time.Time) (string, error) {
	marker := ""
	var bestKey string
	var bestTime time.Time

	for {
		out, err := client.ListObjectsV2(ctx, &tos.ListObjectsV2Input{
			Bucket: bucket,
			ListObjectsInput: tos.ListObjectsInput{
				Prefix:  prefix,
				Marker:  marker,
				MaxKeys: 1000,
				Reverse: true,
			},
		})
		if err != nil {
			return "", err
		}

		for _, obj := range out.Contents {
			key := obj.Key
			if key == "" || key == inputKey || key == outputKey || strings.HasSuffix(key, "/") {
				continue
			}

			lm, err := parseLastModified(obj.LastModified)
			if err != nil {
				continue
			}
			if lm.Before(notBefore) {
				continue
			}

			if bestKey == "" || lm.After(bestTime) {
				bestKey = key
				bestTime = lm
			}
		}

		if !out.IsTruncated || out.NextMarker == "" {
			break
		}
		marker = out.NextMarker
	}

	if bestKey == "" {
		return "", fmt.Errorf("未在TOS中找到转码产物对象")
	}
	return bestKey, nil
}

func renameObject(ctx context.Context, client *tos.ClientV2, bucket, srcKey, dstKey string) error {
	_, err := client.CopyObject(ctx, &tos.CopyObjectInput{
		Bucket:    bucket,
		Key:       dstKey,
		SrcBucket: bucket,
		SrcKey:    srcKey,
	})
	if err != nil {
		return err
	}

	_, err = client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{
		Bucket: bucket,
		Key:    srcKey,
	})
	return err
}

func buildTosPresignedURL(tosCfg TOSConfig, bucket, objectKey string, expiresSeconds int64) (string, error) {
	client, err := newTOSClient(tosCfg)
	if err != nil {
		return "", fmt.Errorf("初始化TOS客户端失败: %v", err)
	}

	resp, err := client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     bucket,
		Key:        strings.TrimPrefix(objectKey, "/"),
		Expires:    expiresSeconds,
	})
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %v", err)
	}
	if resp == nil || resp.SignedUrl == "" {
		return "", fmt.Errorf("生成预签名URL失败: 返回为空")
	}

	return resp.SignedUrl, nil
}

func getTranscodedObjectLocation(config *Config, transcodeInfo *TranscodeInfo) (string, string, error) {
	if transcodeInfo == nil {
		return "", "", fmt.Errorf("转码信息为空")
	}
	storeUri := strings.TrimSpace(transcodeInfo.StoreUri)
	if storeUri == "" {
		return "", "", fmt.Errorf("StoreUri为空")
	}

	if strings.HasPrefix(storeUri, "http://") || strings.HasPrefix(storeUri, "https://") {
		u, err := url.Parse(storeUri)
		if err != nil {
			return "", "", fmt.Errorf("解析StoreUri失败: %v", err)
		}
		host := u.Hostname()
		objectKey := strings.TrimPrefix(u.Path, "/")
		if host == "" || objectKey == "" {
			return "", "", fmt.Errorf("StoreUri缺少主机或路径")
		}
		parts := strings.Split(host, ".")
		bucket := parts[0]
		return bucket, objectKey, nil
	}

	storeUri = strings.TrimPrefix(storeUri, "tos://")
	storeUri = strings.TrimPrefix(storeUri, "/")
	parts := strings.SplitN(storeUri, "/", 2)
	if len(parts) == 1 {
		if config != nil {
			return config.TOS.BucketName, parts[0], nil
		}
		return "", parts[0], nil
	}

	if config != nil {
		if parts[0] == config.TOS.BucketName || strings.HasPrefix(parts[0], "tos-") {
			return parts[0], strings.TrimPrefix(parts[1], "/"), nil
		}
		return config.TOS.BucketName, storeUri, nil
	}

	return parts[0], strings.TrimPrefix(parts[1], "/"), nil
}

func buildTosPublicURL(endpoint, bucket, objectKey string) (string, error) {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return "", fmt.Errorf("endpoint为空")
	}
	if !strings.Contains(ep, "://") {
		ep = "https://" + ep
	}
	u, err := url.Parse(ep)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("endpoint不合法")
	}

	host := u.Host
	if bucket != "" && !strings.HasPrefix(host, bucket+".") {
		host = bucket + "." + host
	}
	u.Host = host
	u.Path = "/" + strings.TrimPrefix(objectKey, "/")
	return u.String(), nil
}

// 上传视频到TOS
func uploadVideoToTOS(config *Config) (string, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤1: 上传视频到TOS桶")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 检查文件是否存在
	if _, err := os.Stat(config.Video.InputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("视频文件不存在: %s", config.Video.InputPath)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(config.Video.InputPath)
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %v", err)
	}
	fileSize := fileInfo.Size()
	fmt.Printf("文件路径: %s\n", config.Video.InputPath)
	fmt.Printf("文件大小: %.2f MB\n", float64(fileSize)/(1024*1024))

	// 生成对象key
	fileName := filepath.Base(config.Video.InputPath)
	objectKey := config.Video.OutputKeyPrefix + fileName
	fmt.Printf("对象Key: %s\n", objectKey)

	// 初始化TOS客户端
	client, err := tos.NewClientV2(
		config.TOS.Endpoint,
		tos.WithRegion(config.TOS.Region),
		tos.WithCredentials(tos.NewStaticCredentials(config.TOS.AccessKey, config.TOS.SecretKey)),
	)
	if err != nil {
		return "", fmt.Errorf("初始化TOS客户端失败: %v", err)
	}

	// 打开文件
	file, err := os.Open(config.Video.InputPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 上传文件并记录时间
	startTime := time.Now()
	ctx := context.Background()

	output, err := client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: config.TOS.BucketName,
			Key:    objectKey,
		},
		Content: file,
	})
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	uploadDuration := time.Since(startTime)
	uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / (1024 * 1024) // MB/s

	fmt.Printf("上传成功!\n")
	fmt.Printf("Request ID: %s\n", output.RequestID)
	fmt.Printf("上传耗时: %.2f 秒\n", uploadDuration.Seconds())
	fmt.Printf("上传速度: %.2f MB/s\n", uploadSpeed)
	fmt.Println()

	return objectKey, nil
}

// 触发工作流转码
func startWorkflow(vodClient VODClientInterface, config *Config, objectKey string) (string, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤2: 触发视频转码工作流")
	fmt.Println("=" + strings.Repeat("=", 60))

	if config.VOD.WorkflowID == "" {
		return "", fmt.Errorf("工作流ID未配置，请在config.yaml中设置vod.workflow_id")
	}

	fmt.Printf("空间名称: %s\n", config.VOD.SpaceName)
	fmt.Printf("工作流ID: %s\n", config.VOD.WorkflowID)
	fmt.Printf("文件路径: %s\n", objectKey)
	fmt.Printf("存储桶: %s\n", config.TOS.BucketName)
	fmt.Println("正在触发工作流...")

	runId, err := vodClient.StartWorkflow(
		config.VOD.SpaceName,
		config.VOD.WorkflowID,
		objectKey,
		config.TOS.BucketName,
	)
	if err != nil {
		return "", fmt.Errorf("触发工作流失败: %v", err)
	}

	fmt.Printf("工作流任务ID (RunId): %s\n", runId)
	fmt.Println()

	return runId, nil
}

// 查询工作流状态
func getWorkflowStatus(vodClient VODClientInterface, runId string) (string, string, *TranscodeInfo, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤3: 查询工作流执行状态")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 轮询查询状态
	maxRetries := 120                 // 最多查询120次（20分钟）
	retryInterval := 10 * time.Second // 每10秒查询一次

	var vid string
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("第 %d 次查询状态...\n", i+1)

		result, err := vodClient.GetWorkflowExecution(runId)
		if err != nil {
			fmt.Printf("查询状态失败: %v，继续重试...\n", err)
			time.Sleep(retryInterval)
			continue
		}

		status := result.Status
		fmt.Printf("当前状态: %s\n", status)

		// 状态说明：
		// "0" - 成功
		// "PendingStart" - 排队中
		// "Running" - 执行中
		// "1xxx" - 用户错误失败
		// "2xxx" - 系统错误失败
		// "Terminated" - 终止

		if status == "0" {
			fmt.Println("转码完成!")
			vid = result.Vid
			workflowVid, transcodeInfo, rErr := vodClient.GetWorkflowExecutionResult(runId)
			if rErr != nil {
				return "", status, nil, fmt.Errorf("获取工作流结果失败: %v", rErr)
			}
			if vid == "" {
				vid = workflowVid
			}
			fmt.Printf("模板ID: %s\n", result.TemplateId)
			fmt.Printf("模板名称: %s\n", result.TemplateName)
			if result.EndTime != "" {
				fmt.Printf("完成时间: %s\n", result.EndTime)
			}
			fmt.Println()
			return vid, status, transcodeInfo, nil
		}

		if status == "PendingStart" || status == "Running" {
			fmt.Printf("转码进行中，等待 %v 后重试...\n", retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// 如果状态是失败
		if strings.HasPrefix(status, "1") || strings.HasPrefix(status, "2") {
			return "", status, nil, fmt.Errorf("转码失败，状态码: %s", status)
		}

		if status == "Terminated" {
			return "", status, nil, fmt.Errorf("转码任务已终止")
		}

		// 未知状态，继续等待
		fmt.Printf("未知状态，等待 %v 后重试...\n", retryInterval)
		time.Sleep(retryInterval)
	}

	return vid, "", nil, fmt.Errorf("转码超时，超过最大重试次数 (%d 次)", maxRetries)
}

// 获取播放地址
func getPlayInfo(vodClient VODClientInterface, config *Config, vid, filePath string, transcodeInfo *TranscodeInfo) (string, *GetPlayInfoResponse, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤4: 获取播放地址")
	fmt.Println("=" + strings.Repeat("=", 60))

	var result *GetPlayInfoResponse
	var err error

	buildResultFromTranscode := func(playUrl string) *GetPlayInfoResponse {
		width, height, bitrate := 1280, 720, 2000000
		size := int64(0)
		duration := float64(0)
		format := "mp4"
		if transcodeInfo != nil {
			if transcodeInfo.Width > 0 {
				width = transcodeInfo.Width
			}
			if transcodeInfo.Height > 0 {
				height = transcodeInfo.Height
			}
			if transcodeInfo.Bitrate > 0 {
				bitrate = transcodeInfo.Bitrate
			}
			if transcodeInfo.Size > 0 {
				size = int64(transcodeInfo.Size)
			}
			if transcodeInfo.Duration > 0 {
				duration = float64(transcodeInfo.Duration)
			}
			if transcodeInfo.Format != "" {
				format = transcodeInfo.Format
			}
		}

		return &GetPlayInfoResponse{
			Vid:        vid,
			Status:     10,
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
					Definition:    "720p",
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
		}
	}

	if transcodeInfo != nil && strings.TrimSpace(transcodeInfo.StoreUri) != "" {
		ctx := context.Background()
		client, cErr := newTOSClient(config.TOS)
		if cErr != nil {
			return "", nil, fmt.Errorf("初始化TOS客户端失败: %v", cErr)
		}

		bucket := config.TOS.BucketName
		outputKey := buildTranscodedOutputKey(filePath)

		fmt.Printf("转码产物StoreUri: %s\n", transcodeInfo.StoreUri)
		parsedBucket, srcKey, locErr := getTranscodedObjectLocation(config, transcodeInfo)
		if locErr != nil {
			return "", nil, fmt.Errorf("解析转码产物位置失败: %v", locErr)
		}
		if parsedBucket != "" {
			bucket = parsedBucket
		}
		if strings.TrimSpace(srcKey) == "" {
			return "", nil, fmt.Errorf("转码产物对象Key为空")
		}
		if srcKey != outputKey {
			if rErr := renameObject(ctx, client, bucket, srcKey, outputKey); rErr != nil {
				if strings.Contains(rErr.Error(), "does not exist") {
					fmt.Printf("重命名对象跳过: %v\n", rErr)
				} else {
					return "", nil, fmt.Errorf("重命名对象失败: %v", rErr)
				}
			}
		}

		playUrl, urlErr := buildTosPresignedURL(config.TOS, bucket, outputKey, 3600)
		if urlErr != nil {
			publicUrl, publicErr := buildTosPublicURL(config.TOS.Endpoint, bucket, outputKey)
			if publicErr != nil {
				return "", nil, fmt.Errorf("生成TOS播放地址失败: %v", urlErr)
			}
			playUrl = publicUrl
		}
		result = buildResultFromTranscode(playUrl)
		err = nil
	}

	if result == nil {
		ctx := context.Background()
		client, cErr := newTOSClient(config.TOS)
		if cErr != nil {
			return "", nil, fmt.Errorf("初始化TOS客户端失败: %v", cErr)
		}

		bucket := config.TOS.BucketName
		outputKey := buildTranscodedOutputKey(filePath)
		prefixCandidates := make([]string, 0, 3)
		if p := path.Dir(filePath); p != "." && p != "/" {
			if !strings.HasSuffix(p, "/") {
				p += "/"
			}
			prefixCandidates = append(prefixCandidates, p)
		}
		if p := strings.TrimSpace(config.Video.OutputKeyPrefix); p != "" {
			if !strings.HasSuffix(p, "/") {
				p += "/"
			}
			prefixCandidates = append(prefixCandidates, p)
		}
		prefixCandidates = append(prefixCandidates, "")

		notBefore := time.Now().Add(-2 * time.Hour)
		var lastErr error
		for attempt := 1; attempt <= 6 && result == nil; attempt++ {
			for _, prefix := range prefixCandidates {
				srcKey, fErr := findTranscodedSourceObjectKey(ctx, client, bucket, prefix, filePath, outputKey, notBefore)
				if fErr != nil {
					lastErr = fErr
					continue
				}
				if srcKey != outputKey {
					if rErr := renameObject(ctx, client, bucket, srcKey, outputKey); rErr != nil {
						return "", nil, fmt.Errorf("重命名对象失败: %v", rErr)
					}
				}

				playUrl, urlErr := buildTosPresignedURL(config.TOS, bucket, outputKey, 3600)
				if urlErr != nil {
					publicUrl, publicErr := buildTosPublicURL(config.TOS.Endpoint, bucket, outputKey)
					if publicErr != nil {
						return "", nil, fmt.Errorf("生成TOS播放地址失败: %v", urlErr)
					}
					playUrl = publicUrl
				}
				result = buildResultFromTranscode(playUrl)
				err = nil
				break
			}
			if result != nil {
				break
			}
			if attempt < 6 {
				fmt.Printf("未定位到转码产物，等待5秒后重试（%d/6）...\n", attempt)
				time.Sleep(5 * time.Second)
			}
		}
		if result == nil && lastErr != nil {
			fmt.Printf("定位转码产物失败: %v\n", lastErr)
		}
	}

	if result == nil && vid != "" {
		fmt.Printf("请求格式: mp4\n")
		fmt.Printf("请求清晰度: 720p\n")
		fmt.Println("正在通过Vid获取播放地址...")

		result, err = vodClient.GetPlayInfo(vid, "mp4", "720p")
		if err != nil {
			return "", nil, fmt.Errorf("获取播放地址失败: %v", err)
		}
	} else if result == nil {
		fmt.Println("未获取到转码产物信息，无法获取播放地址")
		fmt.Printf("文件路径: %s\n", filePath)
		return "", nil, fmt.Errorf("未获取到转码产物信息")
	}

	if result.Status != 10 {
		return "", nil, fmt.Errorf("视频状态异常，状态码: %d (10表示成功)", result.Status)
	}

	if len(result.PlayInfoList) == 0 {
		return "", nil, fmt.Errorf("未找到720p格式的播放地址，请检查转码是否成功")
	}

	playInfo := result.PlayInfoList[0]
	playUrl := playInfo.MainPlayUrl

	fmt.Printf("播放地址获取成功!\n")
	fmt.Printf("主播放地址: %s\n", playUrl)
	if playInfo.BackupPlayUrl != "" {
		fmt.Printf("备播放地址: %s\n", playInfo.BackupPlayUrl)
	}
	fmt.Printf("视频格式: %s\n", playInfo.Format)
	fmt.Printf("编码格式: %s\n", playInfo.Codec)
	fmt.Printf("清晰度: %s\n", playInfo.Definition)
	fmt.Printf("分辨率: %dx%d\n", playInfo.Width, playInfo.Height)
	fmt.Printf("码率: %d bps\n", playInfo.Bitrate)
	fmt.Printf("文件大小: %.2f MB\n", float64(playInfo.Size)/(1024*1024))
	fmt.Println()

	return playUrl, result, nil
}

// 显示测试结果
func displayTestResults(config *Config, uploadDuration time.Duration, fileSize int64, playInfo *GetPlayInfoResponse) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("测试结果汇总")
	fmt.Println("=" + strings.Repeat("=", 60))

	uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / (1024 * 1024)
	fmt.Printf("上传速度: %.2f MB/s\n", uploadSpeed)
	fmt.Printf("上传耗时: %.2f 秒\n", uploadDuration.Seconds())

	if playInfo != nil && len(playInfo.PlayInfoList) > 0 {
		transcodedSize := playInfo.PlayInfoList[0].Size
		compressionRatio := float64(transcodedSize) / float64(fileSize) * 100
		fmt.Printf("原始文件大小: %.2f MB\n", float64(fileSize)/(1024*1024))
		fmt.Printf("转码后文件大小: %.2f MB\n", float64(transcodedSize)/(1024*1024))
		fmt.Printf("压缩比: %.2f%%\n", compressionRatio)
		fmt.Printf("转码后分辨率: %dx%d\n", playInfo.PlayInfoList[0].Width, playInfo.PlayInfoList[0].Height)
		fmt.Printf("转码后码率: %d bps\n", playInfo.PlayInfoList[0].Bitrate)
	}
	fmt.Println("播放器时延: 需要实际播放测试")
	fmt.Println()
}

func main() {
	fmt.Println("大疆云相册 - 视频上传转码Demo")
	fmt.Println()

	// 加载配置
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 验证必要配置
	if config.TOS.AccessKey == "" || config.TOS.SecretKey == "" {
		fmt.Fprintf(os.Stderr, "错误: TOS AccessKey或SecretKey未配置\n")
		fmt.Fprintf(os.Stderr, "请在config.yaml中配置或设置环境变量ACCESS_KEY和SECRET_KEY\n")
		os.Exit(1)
	}

	if config.VOD.AccessKey == "" || config.VOD.SecretKey == "" {
		fmt.Fprintf(os.Stderr, "错误: VOD AccessKey或SecretKey未配置\n")
		fmt.Fprintf(os.Stderr, "请在config.yaml中配置或设置环境变量ACCESS_KEY和SECRET_KEY\n")
		os.Exit(1)
	}

	// 创建VOD客户端（使用官方SDK V1.0）
	// 注意：使用V1.0 SDK，因为V2.0 SDK目前不支持StartWorkflow API
	vodClient := NewVODClient(
		config.VOD.AccessKey,
		config.VOD.SecretKey,
		config.VOD.Region,
		config.VOD.APIEndpoint,
	)
	if vodClient == nil {
		fmt.Fprintf(os.Stderr, "创建VOD客户端失败\n")
		os.Exit(1)
	}

	// 步骤1: 上传视频到TOS
	startTime := time.Now()
	objectKey, err := uploadVideoToTOS(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "上传失败: %v\n", err)
		os.Exit(1)
	}
	uploadDuration := time.Since(startTime)

	// 获取文件大小用于测试结果
	fileInfo, _ := os.Stat(config.Video.InputPath)
	fileSize := fileInfo.Size()

	// 步骤2: 触发转码工作流
	fmt.Println("\n注意: 如果遇到签名错误，建议使用火山引擎官方VOD SDK\n参考: https://www.volcengine.com/docs/4/3479")

	runId, err := startWorkflow(vodClient, config, objectKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n触发工作流失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n提示: 如果遇到签名认证错误，请考虑:\n")
		fmt.Fprintf(os.Stderr, "1. 使用火山引擎官方VOD SDK\n")
		fmt.Fprintf(os.Stderr, "2. 检查AccessKey和SecretKey是否正确\n")
		fmt.Fprintf(os.Stderr, "3. 参考官方文档: https://www.volcengine.com/docs/4/3479\n\n")
		os.Exit(1)
	}

	// 步骤3: 查询转码状态
	vid, _, transcodeInfo, err := getWorkflowStatus(vodClient, runId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询转码状态失败: %v\n", err)
		os.Exit(1)
	}

	// 步骤4: 获取播放地址
	// 如果Vid为空，尝试通过文件路径获取播放地址
	playUrl, playInfo, err := getPlayInfo(vodClient, config, vid, objectKey, transcodeInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取播放地址失败: %v\n", err)
		os.Exit(1)
	}

	// 显示测试结果
	displayTestResults(config, uploadDuration, fileSize, playInfo)

	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("Demo执行完成!")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("播放地址: %s\n", playUrl)
	fmt.Println()
	fmt.Println("您可以使用以下方式播放视频:")
	fmt.Printf("1. 浏览器直接访问: %s\n", playUrl)
	fmt.Println("2. 使用HTML5 video标签:")
	fmt.Printf("   <video controls><source src=\"%s\" type=\"video/mp4\"></video>\n", playUrl)
	fmt.Println()
}
