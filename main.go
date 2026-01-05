package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	TOS  TOSConfig  `yaml:"tos"`
	VOD  VODConfig  `yaml:"vod"`
	Video VideoConfig `yaml:"video"`
}

type TOSConfig struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
}

type VODConfig struct {
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	Region     string `yaml:"region"`
	SpaceName  string `yaml:"space_name"`
	WorkflowID string `yaml:"workflow_id"`
	APIEndpoint string `yaml:"api_endpoint"`
}

type VideoConfig struct {
	InputPath      string `yaml:"input_path"`
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
	// 如果TOS和VOD使用相同的AK/SK，优先使用统一的ACCESS_KEY和SECRET_KEY
	accessKey := os.Getenv("ACCESS_KEY")
	secretKey := os.Getenv("SECRET_KEY")
	
	// TOS配置
	if envVal := os.Getenv("TOS_ACCESS_KEY"); envVal != "" {
		config.TOS.AccessKey = envVal
	} else if accessKey != "" {
		config.TOS.AccessKey = accessKey
	}
	if envVal := os.Getenv("TOS_SECRET_KEY"); envVal != "" {
		config.TOS.SecretKey = envVal
	} else if secretKey != "" {
		config.TOS.SecretKey = secretKey
	}
	
	// VOD配置
	if envVal := os.Getenv("VOD_ACCESS_KEY"); envVal != "" {
		config.VOD.AccessKey = envVal
	} else if accessKey != "" {
		config.VOD.AccessKey = accessKey
	}
	if envVal := os.Getenv("VOD_SECRET_KEY"); envVal != "" {
		config.VOD.SecretKey = envVal
	} else if secretKey != "" {
		config.VOD.SecretKey = secretKey
	}
	if envVal := os.Getenv("VOD_WORKFLOW_ID"); envVal != "" {
		config.VOD.WorkflowID = envVal
	}

	return &config, nil
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
func getWorkflowStatus(vodClient VODClientInterface, runId string) (string, string, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤3: 查询工作流执行状态")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 轮询查询状态
	maxRetries := 120 // 最多查询120次（20分钟）
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
			fmt.Printf("视频ID (Vid): %s\n", vid)
			fmt.Printf("模板ID: %s\n", result.TemplateId)
			fmt.Printf("模板名称: %s\n", result.TemplateName)
			if result.EndTime != "" {
				fmt.Printf("完成时间: %s\n", result.EndTime)
			}
			
			// 如果Vid为空，尝试从GetWorkflowExecutionResult获取
			if vid == "" {
				fmt.Println("Vid为空，尝试从执行结果获取...")
				if clientV1, ok := vodClient.(interface {
					GetWorkflowExecutionResult(runId string) (string, error)
				}); ok {
					if resultVid, err := clientV1.GetWorkflowExecutionResult(runId); err == nil {
						vid = resultVid
						fmt.Printf("从执行结果获取到Vid: %s\n", vid)
					} else {
						fmt.Printf("从执行结果获取Vid失败: %v\n", err)
					}
				}
			}
			
			fmt.Println()
			return vid, status, nil
		}

		if status == "PendingStart" || status == "Running" {
			fmt.Printf("转码进行中，等待 %v 后重试...\n", retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// 如果状态是失败
		if strings.HasPrefix(status, "1") || strings.HasPrefix(status, "2") {
			return "", status, fmt.Errorf("转码失败，状态码: %s", status)
		}

		if status == "Terminated" {
			return "", status, fmt.Errorf("转码任务已终止")
		}

		// 未知状态，继续等待
		fmt.Printf("未知状态，等待 %v 后重试...\n", retryInterval)
		time.Sleep(retryInterval)
	}

	return vid, "", fmt.Errorf("转码超时，超过最大重试次数 (%d 次)", maxRetries)
}

// 获取播放地址
func getPlayInfo(vodClient VODClientInterface, vid string) (string, *GetPlayInfoResponse, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤4: 获取播放地址")
	fmt.Println("=" + strings.Repeat("=", 60))

	fmt.Printf("视频ID (Vid): %s\n", vid)
	fmt.Printf("请求格式: mp4\n")
	fmt.Printf("请求清晰度: 720p\n")
	fmt.Println("正在获取播放地址...")

	result, err := vodClient.GetPlayInfo(vid, "mp4", "720p")
	if err != nil {
		return "", nil, fmt.Errorf("获取播放地址失败: %v", err)
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
		fmt.Fprintf(os.Stderr, "请在config.yaml中配置或设置环境变量TOS_ACCESS_KEY和TOS_SECRET_KEY\n")
		os.Exit(1)
	}

	if config.VOD.AccessKey == "" || config.VOD.SecretKey == "" {
		fmt.Fprintf(os.Stderr, "错误: VOD AccessKey或SecretKey未配置\n")
		fmt.Fprintf(os.Stderr, "请在config.yaml中配置或设置环境变量VOD_ACCESS_KEY和VOD_SECRET_KEY\n")
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
	fmt.Println("\n注意: 如果遇到签名错误，建议使用火山引擎官方VOD SDK")
	fmt.Println("参考: https://www.volcengine.com/docs/4/3479\n")
	
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
	vid, _, err := getWorkflowStatus(vodClient, runId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询转码状态失败: %v\n", err)
		os.Exit(1)
	}

	// 步骤4: 获取播放地址
	if vid == "" {
		fmt.Fprintf(os.Stderr, "警告: 未获取到Vid，无法获取播放地址\n")
		fmt.Fprintf(os.Stderr, "提示: DirectUrl模式下，可能需要通过其他方式获取Vid\n")
		fmt.Fprintf(os.Stderr, "您可以:\n")
		fmt.Fprintf(os.Stderr, "1. 检查工作流执行结果中的其他字段\n")
		fmt.Fprintf(os.Stderr, "2. 通过TOS文件路径查询对应的Vid\n")
		fmt.Fprintf(os.Stderr, "3. 使用控制台查看转码后的视频信息\n")
		os.Exit(1)
	}
	
	playUrl, playInfo, err := getPlayInfo(vodClient, vid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取播放地址失败: %v\n", err)
		os.Exit(1)
	}

	// 显示测试结果
	displayTestResults(config, uploadDuration, fileSize, playInfo)

	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("Demo执行完成!")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("视频ID (Vid): %s\n", vid)
	fmt.Printf("播放地址: %s\n", playUrl)
	fmt.Println()
	fmt.Println("您可以使用以下方式播放视频:")
	fmt.Printf("1. 浏览器直接访问: %s\n", playUrl)
	fmt.Println("2. 使用HTML5 video标签:")
	fmt.Printf("   <video controls><source src=\"%s\" type=\"video/mp4\"></video>\n", playUrl)
	fmt.Println()
}

