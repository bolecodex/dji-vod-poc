package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
	"gopkg.in/yaml.v3"
)

var vmafSupportChecked bool
var vmafSupported bool
var vmafSupportErr error

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
			if value != "" && value[0] != '\'' && value[0] != '"' {
				for i, r := range value {
					if r == '#' {
						if i > 0 && unicode.IsSpace(rune(value[i-1])) {
							value = strings.TrimSpace(value[:i-1])
						}
						break
					}
				}
			}
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

func buildTranscodedOutputKey(inputObjectKey string, transcodeInfo *TranscodeInfo) string {
	dir := path.Dir(inputObjectKey)
	base := path.Base(inputObjectKey)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		stem = base
	}

	label := "720P"
	if transcodeInfo != nil {
		dim := transcodeInfo.Height
		if transcodeInfo.Width > 0 && transcodeInfo.Height > 0 {
			if transcodeInfo.Width < transcodeInfo.Height {
				dim = transcodeInfo.Width
			}
		}
		if dim > 0 {
			label = fmt.Sprintf("%dP", dim)
		}
	}

	outName := stem + "-转码后-" + label + ".mp4"
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

func objectExists(ctx context.Context, client *tos.ClientV2, bucket, objectKey string) (bool, error) {
	_, err := client.HeadObjectV2(ctx, &tos.HeadObjectV2Input{
		Bucket: bucket,
		Key:    strings.TrimPrefix(objectKey, "/"),
	})
	if err == nil {
		return true, nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "invalid body") {
		return false, nil
	}
	if strings.Contains(msg, "nosuchkey") || strings.Contains(msg, "no such key") || strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") {
		return false, nil
	}
	return false, err
}

func isHLSOutput(transcodeInfo *TranscodeInfo) bool {
	if transcodeInfo == nil {
		return false
	}
	format := strings.ToLower(strings.TrimSpace(transcodeInfo.Format))
	store := strings.ToLower(strings.TrimSpace(transcodeInfo.StoreUri))
	return format == "hls" || strings.HasSuffix(store, ".m3u8")
}

func buildTempWorkflowInputKey(originalObjectKey string) string {
	dir := path.Dir(originalObjectKey)
	base := path.Base(originalObjectKey)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		stem = base
	}
	name := fmt.Sprintf("%s-wf-%s%s", stem, time.Now().Format("20060102T150405"), ext)
	if dir == "." || dir == "/" {
		return name
	}
	return path.Join(dir, name)
}

func ensureHLSOutputAvailable(vodClient VODClientInterface, config *Config, inputObjectKey, vid string, transcodeInfo *TranscodeInfo) (string, *TranscodeInfo, error) {
	if !isHLSOutput(transcodeInfo) {
		return vid, transcodeInfo, nil
	}

	ctx := context.Background()
	client, err := newTOSClient(config.TOS)
	if err != nil {
		return vid, transcodeInfo, fmt.Errorf("初始化TOS客户端失败: %v", err)
	}

	bucket, srcKey, locErr := getTranscodedObjectLocation(config, transcodeInfo)
	if locErr == nil && bucket != "" && srcKey != "" {
		ok, exErr := objectExists(ctx, client, bucket, srcKey)
		if exErr != nil {
			return vid, transcodeInfo, fmt.Errorf("检查HLS产物是否存在失败: %v", exErr)
		}
		if ok {
			return vid, transcodeInfo, nil
		}
	}

	if strings.TrimSpace(inputObjectKey) == "" {
		return vid, transcodeInfo, fmt.Errorf("HLS产物不存在，且输入对象Key为空，无法重新触发工作流")
	}

	if config.VOD.WorkflowID == "" {
		return vid, transcodeInfo, fmt.Errorf("工作流ID未配置，无法重新触发工作流")
	}

	for attempt := 1; attempt <= 2; attempt++ {
		tempKey := buildTempWorkflowInputKey(inputObjectKey)
		fmt.Printf("HLS产物不存在，重新触发工作流（%d/2），输入对象: %s\n", attempt, tempKey)

		_, cpErr := client.CopyObject(ctx, &tos.CopyObjectInput{
			Bucket:    config.TOS.BucketName,
			Key:       strings.TrimPrefix(tempKey, "/"),
			SrcBucket: config.TOS.BucketName,
			SrcKey:    strings.TrimPrefix(inputObjectKey, "/"),
		})
		if cpErr != nil {
			return vid, transcodeInfo, fmt.Errorf("复制输入对象失败: %v", cpErr)
		}

		runId, sErr := startWorkflow(vodClient, config, tempKey)
		if sErr != nil {
			return vid, transcodeInfo, sErr
		}

		newVid, _, newTrans, wErr := getWorkflowStatus(vodClient, runId)
		if wErr != nil {
			return vid, transcodeInfo, wErr
		}

		b2, k2, locErr2 := getTranscodedObjectLocation(config, newTrans)
		if locErr2 == nil && b2 != "" && k2 != "" {
			ok, exErr := objectExists(ctx, client, b2, k2)
			if exErr != nil {
				return newVid, newTrans, fmt.Errorf("检查HLS产物是否存在失败: %v", exErr)
			}
			if ok {
				return newVid, newTrans, nil
			}
		}
	}

	return vid, transcodeInfo, fmt.Errorf("HLS产物不存在：StoreUri=%s", strings.TrimSpace(transcodeInfo.StoreUri))
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

func vmafSampleSeconds() int {
	value := strings.TrimSpace(os.Getenv("VMAF_SECONDS"))
	if value == "" {
		return 30
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 30
	}
	if n < 0 {
		return 30
	}
	return n
}

func checkVMAFSupport() (bool, error) {
	if vmafSupportChecked {
		return vmafSupported, vmafSupportErr
	}
	vmafSupportChecked = true

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		vmafSupportErr = fmt.Errorf("未找到ffmpeg")
		return false, vmafSupportErr
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		vmafSupportErr = fmt.Errorf("未找到ffprobe")
		return false, vmafSupportErr
	}

	out, err := exec.Command("ffmpeg", "-hide_banner", "-filters").CombinedOutput()
	if err != nil {
		vmafSupportErr = fmt.Errorf("检查ffmpeg filters失败: %v", err)
		return false, vmafSupportErr
	}
	if !bytes.Contains(bytes.ToLower(out), []byte("libvmaf")) {
		vmafSupportErr = fmt.Errorf("ffmpeg缺少libvmaf")
		return false, vmafSupportErr
	}

	vmafSupported = true
	return true, nil
}

func probeVideoDimensions(input string) (int, int, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		input,
	).CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe失败: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("ffprobe输出不合法: %s", strings.TrimSpace(string(out)))
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("解析宽度失败: %v", err)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("解析高度失败: %v", err)
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("分辨率不合法: %dx%d", w, h)
	}
	return w, h, nil
}

func computeVMAF(referencePath, distortedInput string, width, height int, sampleSeconds int) (float64, error) {
	ok, err := checkVMAFSupport()
	if !ok {
		return 0, err
	}
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("分辨率不合法: %dx%d", width, height)
	}
	if strings.TrimSpace(referencePath) == "" || strings.TrimSpace(distortedInput) == "" {
		return 0, fmt.Errorf("参考或对比输入为空")
	}

	tmpDir, err := os.MkdirTemp("", "vmaf-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "vmaf.json")
	filter := fmt.Sprintf(
		"[0:v]setpts=PTS-STARTPTS,scale=%d:%d:flags=bicubic[ref];[1:v]setpts=PTS-STARTPTS,scale=%d:%d:flags=bicubic[dist];[dist][ref]libvmaf=log_fmt=json:log_path=%s",
		width, height, width, height, logPath,
	)

	args := []string{"-hide_banner", "-y", "-i", referencePath}
	if strings.Contains(strings.ToLower(distortedInput), ".m3u8") {
		args = append(args,
			"-protocol_whitelist", "file,http,https,tcp,tls,crypto",
			"-allowed_extensions", "ALL",
		)
	}
	args = append(args, "-i", distortedInput)
	if sampleSeconds > 0 {
		args = append(args, "-t", strconv.Itoa(sampleSeconds))
	}
	args = append(args, "-an", "-sn", "-lavfi", filter, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("ffmpeg计算VMAF失败: %v", strings.TrimSpace(string(out)))
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		return 0, fmt.Errorf("读取VMAF日志失败: %v", err)
	}

	var parsed struct {
		PooledMetrics map[string]struct {
			Mean float64 `json:"mean"`
		} `json:"pooled_metrics"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, fmt.Errorf("解析VMAF日志失败: %v", err)
	}
	metric, ok := parsed.PooledMetrics["vmaf"]
	if !ok {
		return 0, fmt.Errorf("VMAF日志缺少pooled_metrics.vmaf")
	}
	return metric.Mean, nil
}

func fetchTextFromURL(raw string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseBucketAndObjectKeyFromURL(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	host := strings.TrimSpace(u.Hostname())
	key := strings.TrimPrefix(u.Path, "/")
	if host == "" || key == "" {
		return "", "", fmt.Errorf("URL缺少host或path")
	}
	parts := strings.Split(host, ".")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("无法从URL解析bucket")
	}
	return parts[0], key, nil
}

func rewriteExtXKeyLine(config *Config, bucket, playlistDir, line string, expiresSeconds int64) (string, error) {
	idx := strings.Index(line, "URI=\"")
	if idx < 0 {
		return line, nil
	}
	start := idx + len("URI=\"")
	end := strings.Index(line[start:], "\"")
	if end < 0 {
		return line, nil
	}
	uri := line[start : start+end]
	if uri == "" {
		return line, nil
	}
	if strings.Contains(uri, "://") {
		return line, nil
	}
	pathPart := uri
	if q := strings.IndexByte(pathPart, '?'); q >= 0 {
		pathPart = pathPart[:q]
	}
	key := path.Clean(path.Join(playlistDir, pathPart))
	signed, err := buildTosPresignedURL(config.TOS, bucket, key, expiresSeconds)
	if err != nil {
		return "", err
	}
	newLine := line[:start] + signed + line[start+end:]
	return newLine, nil
}

func rewriteMediaPlaylist(config *Config, bucket, playlistKey, content string, expiresSeconds int64) (string, error) {
	playlistDir := path.Dir(playlistKey)
	lines := strings.Split(content, "\n")
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-KEY") {
			updated, err := rewriteExtXKeyLine(config, bucket, playlistDir, lines[i], expiresSeconds)
			if err != nil {
				return "", err
			}
			lines[i] = updated
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "://") {
			continue
		}
		pathPart := line
		if q := strings.IndexByte(pathPart, '?'); q >= 0 {
			pathPart = pathPart[:q]
		}
		segmentKey := path.Clean(path.Join(playlistDir, pathPart))
		signed, err := buildTosPresignedURL(config.TOS, bucket, segmentKey, expiresSeconds)
		if err != nil {
			return "", err
		}
		lines[i] = signed
	}
	return strings.Join(lines, "\n"), nil
}

func prepareHLSPlaylistForFFmpeg(config *Config, playlistURL string, expiresSeconds int64) (string, func(), error) {
	bucket, playlistKey, err := parseBucketAndObjectKeyFromURL(playlistURL)
	if err != nil {
		bucket = strings.TrimSpace(config.TOS.BucketName)
	}
	if bucket == "" {
		return "", nil, fmt.Errorf("bucket为空")
	}
	if playlistKey == "" {
		_, k, kErr := parseBucketAndObjectKeyFromURL(playlistURL)
		if kErr != nil {
			return "", nil, kErr
		}
		playlistKey = k
	}

	rootContent, err := fetchTextFromURL(playlistURL)
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "hls-vmaf-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	writeFile := func(name string, text string) (string, error) {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			return "", err
		}
		return p, nil
	}

	isMaster := strings.Contains(rootContent, "#EXT-X-STREAM-INF")
	if !isMaster {
		rewritten, err := rewriteMediaPlaylist(config, bucket, playlistKey, rootContent, expiresSeconds)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		p, err := writeFile("root.m3u8", rewritten)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		return p, cleanup, nil
	}

	rootDirKey := path.Dir(playlistKey)
	lines := strings.Split(rootContent, "\n")
	variantIndex := 0
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "://") {
			continue
		}
		pathPart := line
		if q := strings.IndexByte(pathPart, '?'); q >= 0 {
			pathPart = pathPart[:q]
		}
		variantKey := path.Clean(path.Join(rootDirKey, pathPart))
		variantURL, err := buildTosPresignedURL(config.TOS, bucket, variantKey, expiresSeconds)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		variantContent, err := fetchTextFromURL(variantURL)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		rewrittenVariant, err := rewriteMediaPlaylist(config, bucket, variantKey, variantContent, expiresSeconds)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		localVariantName := fmt.Sprintf("variant-%d.m3u8", variantIndex)
		variantIndex++
		localVariantPath, err := writeFile(localVariantName, rewrittenVariant)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		lines[i] = localVariantPath
	}

	rootPath, err := writeFile("master.m3u8", strings.Join(lines, "\n"))
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return rootPath, cleanup, nil
}

// 上传视频到TOS
func uploadVideoToTOS(config *Config, inputPath string) (string, int64, time.Duration, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤1: 上传视频到TOS桶")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 检查文件是否存在
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return "", 0, 0, fmt.Errorf("视频文件不存在: %s", inputPath)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("获取文件信息失败: %v", err)
	}
	fileSize := fileInfo.Size()
	fmt.Printf("文件路径: %s\n", inputPath)
	fmt.Printf("文件大小: %.2f MB\n", float64(fileSize)/(1024*1024))

	// 生成对象key
	fileName := filepath.Base(inputPath)
	objectKey := config.Video.OutputKeyPrefix + fileName
	fmt.Printf("对象Key: %s\n", objectKey)

	// 初始化TOS客户端
	client, err := newTOSClient(config.TOS)
	if err != nil {
		return "", 0, 0, fmt.Errorf("初始化TOS客户端失败: %v", err)
	}

	ctx := context.Background()
	exists, exErr := objectExists(ctx, client, config.TOS.BucketName, objectKey)
	if exErr != nil {
		return "", 0, 0, fmt.Errorf("检查对象是否已存在失败: %v", exErr)
	}
	if exists {
		fmt.Println("对象已存在，跳过上传")
		fmt.Println()
		return objectKey, fileSize, 0, nil
	}

	// 打开文件
	file, err := os.Open(inputPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 上传文件并记录时间
	startTime := time.Now()

	const multipartThreshold = int64(128 * 1024 * 1024)
	const partSize = int64(20 * 1024 * 1024)

	if fileSize >= multipartThreshold {
		createOut, err := client.CreateMultipartUploadV2(ctx, &tos.CreateMultipartUploadV2Input{
			Bucket: config.TOS.BucketName,
			Key:    objectKey,
		})
		if err != nil {
			return "", 0, 0, fmt.Errorf("初始化分片上传失败: %v", err)
		}

		offset := int64(0)
		partNumber := 1
		parts := make([]tos.UploadedPartV2, 0, 32)
		for offset < fileSize {
			uploadSize := partSize
			if fileSize-offset < partSize {
				uploadSize = fileSize - offset
			}

			var partOut *tos.UploadPartV2Output
			var lastErr error
			for attempt := 1; attempt <= 5; attempt++ {
				if _, err := file.Seek(offset, io.SeekStart); err != nil {
					lastErr = err
					break
				}

				out, err := client.UploadPartV2(ctx, &tos.UploadPartV2Input{
					UploadPartBasicInput: tos.UploadPartBasicInput{
						Bucket:     config.TOS.BucketName,
						Key:        objectKey,
						UploadID:   createOut.UploadID,
						PartNumber: partNumber,
					},
					Content:       io.LimitReader(file, uploadSize),
					ContentLength: uploadSize,
				})
				if err == nil {
					partOut = out
					lastErr = nil
					break
				}
				lastErr = err
				if attempt < 5 {
					newClient, cErr := newTOSClient(config.TOS)
					if cErr == nil {
						client = newClient
					}
					time.Sleep(time.Duration(attempt*2) * time.Second)
				}
			}
			if lastErr != nil {
				_ = file.Close()
				_, _ = client.AbortMultipartUpload(ctx, &tos.AbortMultipartUploadInput{
					Bucket:   config.TOS.BucketName,
					Key:      objectKey,
					UploadID: createOut.UploadID,
				})
				if lastErr == io.EOF {
					return "", 0, 0, fmt.Errorf("读取分片数据失败: %v", lastErr)
				}
				if strings.Contains(lastErr.Error(), "seek") {
					return "", 0, 0, fmt.Errorf("定位文件分片失败: %v", lastErr)
				}
				return "", 0, 0, fmt.Errorf("上传分片失败: %v", lastErr)
			}
			parts = append(parts, tos.UploadedPartV2{PartNumber: partNumber, ETag: partOut.ETag})

			offset += uploadSize
			partNumber++
		}

		completeOut, err := client.CompleteMultipartUploadV2(ctx, &tos.CompleteMultipartUploadV2Input{
			Bucket:   config.TOS.BucketName,
			Key:      objectKey,
			UploadID: createOut.UploadID,
			Parts:    parts,
		})
		if err != nil {
			return "", 0, 0, fmt.Errorf("合并分片失败: %v", err)
		}

		uploadDuration := time.Since(startTime)
		uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / (1024 * 1024)
		fmt.Printf("上传成功!\n")
		fmt.Printf("Request ID: %s\n", completeOut.RequestID)
		fmt.Printf("上传耗时: %.2f 秒\n", uploadDuration.Seconds())
		fmt.Printf("上传速度: %.2f MB/s\n", uploadSpeed)
		fmt.Println()

		return objectKey, fileSize, uploadDuration, nil
	}

	output, err := client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: config.TOS.BucketName,
			Key:    objectKey,
		},
		Content: file,
	})
	if err != nil {
		return "", 0, 0, fmt.Errorf("上传文件失败: %v", err)
	}

	uploadDuration := time.Since(startTime)
	uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / (1024 * 1024) // MB/s

	fmt.Printf("上传成功!\n")
	fmt.Printf("Request ID: %s\n", output.RequestID)
	fmt.Printf("上传耗时: %.2f 秒\n", uploadDuration.Seconds())
	fmt.Printf("上传速度: %.2f MB/s\n", uploadSpeed)
	fmt.Println()

	return objectKey, fileSize, uploadDuration, nil
}

func listVideoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	allowed := map[string]struct{}{
		".mp4":  {},
		".mov":  {},
		".mkv":  {},
		".avi":  {},
		".m4v":  {},
		".flv":  {},
		".webm": {},
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := allowed[ext]; !ok {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}

	return paths, nil
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
	startTime := time.Now()

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
	fmt.Printf("转码任务提交时间: %s\n", startTime.Format(time.RFC3339))
	fmt.Println()

	return runId, nil
}

// 查询工作流状态
func getWorkflowStatus(vodClient VODClientInterface, runId string) (string, string, *TranscodeInfo, error) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("步骤3: 查询工作流执行状态")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 轮询查询状态
	maxRetries := 720
	retryInterval := 10 * time.Second

	localStart := time.Now()
	var vid string
	var startTimeStr string
	var startTime time.Time
	var startTimeOk bool
	var printedStart bool
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
		if !printedStart {
			if result.StartTime != "" {
				startTimeStr = result.StartTime
				if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
					startTime = t
					startTimeOk = true
				}
				fmt.Printf("转码开始时间: %s\n", startTimeStr)
				printedStart = true
			} else if status == "Running" {
				startTime = localStart
				startTimeOk = true
				fmt.Printf("转码开始时间: %s\n", startTime.Format(time.RFC3339))
				printedStart = true
			}
		}

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
			endTimeStr := result.EndTime
			if endTimeStr == "" {
				endTimeStr = time.Now().Format(time.RFC3339)
			}
			fmt.Printf("转码结束时间: %s\n", endTimeStr)
			if !printedStart {
				fmt.Printf("转码开始时间: %s\n", localStart.Format(time.RFC3339))
				startTime = localStart
				startTimeOk = true
				printedStart = true
			}
			if startTimeOk {
				if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
					fmt.Printf("转码耗时: %s\n", endTime.Sub(startTime).Truncate(time.Second))
				} else {
					fmt.Printf("转码耗时: %s\n", time.Since(startTime).Truncate(time.Second))
				}
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
		definition := "720p"
		if transcodeInfo != nil {
			if transcodeInfo.Width > 0 {
				width = transcodeInfo.Width
			}
			if transcodeInfo.Height > 0 {
				height = transcodeInfo.Height
				dim := transcodeInfo.Height
				if transcodeInfo.Width > 0 && transcodeInfo.Height > 0 {
					if transcodeInfo.Width < transcodeInfo.Height {
						dim = transcodeInfo.Width
					}
				}
				definition = fmt.Sprintf("%dp", dim)
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
		}
	}

	isHLS := transcodeInfo != nil && (strings.EqualFold(strings.TrimSpace(transcodeInfo.Format), "hls") || strings.HasSuffix(strings.ToLower(strings.TrimSpace(transcodeInfo.StoreUri)), ".m3u8"))
	if transcodeInfo != nil && strings.TrimSpace(transcodeInfo.StoreUri) != "" {
		ctx := context.Background()
		client, cErr := newTOSClient(config.TOS)
		if cErr != nil {
			return "", nil, fmt.Errorf("初始化TOS客户端失败: %v", cErr)
		}

		bucket := config.TOS.BucketName
		outputKey := buildTranscodedOutputKey(filePath, transcodeInfo)

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
		if isHLS {
			outputKey = srcKey
		} else {
			if srcKey != outputKey {
				if rErr := renameObject(ctx, client, bucket, srcKey, outputKey); rErr != nil {
					if strings.Contains(rErr.Error(), "does not exist") {
						fmt.Printf("重命名对象跳过: %v\n", rErr)
					} else {
						return "", nil, fmt.Errorf("重命名对象失败: %v", rErr)
					}
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
		outputKey := buildTranscodedOutputKey(filePath, transcodeInfo)
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
func displayTestResults(config *Config, uploadDuration time.Duration, fileSize int64, inputPath string, playURL string, playInfo *GetPlayInfoResponse) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("测试结果汇总")
	fmt.Println("=" + strings.Repeat("=", 60))

	if uploadDuration > 0 {
		uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / (1024 * 1024)
		fmt.Printf("上传速度: %.2f MB/s\n", uploadSpeed)
		fmt.Printf("上传耗时: %.2f 秒\n", uploadDuration.Seconds())
	} else {
		fmt.Printf("上传速度: -\n")
		fmt.Printf("上传耗时: -\n")
	}

	if playInfo != nil && len(playInfo.PlayInfoList) > 0 {
		transcodedSize := playInfo.PlayInfoList[0].Size
		compressionRatio := (1 - float64(transcodedSize)/float64(fileSize)) * 100
		fmt.Printf("原始文件大小: %.2f MB\n", float64(fileSize)/(1024*1024))
		fmt.Printf("转码后文件大小: %.2f MB\n", float64(transcodedSize)/(1024*1024))
		fmt.Printf("压缩比: %.2f%%\n", compressionRatio)
		fmt.Printf("转码后分辨率: %dx%d\n", playInfo.PlayInfoList[0].Width, playInfo.PlayInfoList[0].Height)
		fmt.Printf("转码后码率: %d bps\n", playInfo.PlayInfoList[0].Bitrate)

		w, h := playInfo.PlayInfoList[0].Width, playInfo.PlayInfoList[0].Height
		if w <= 0 || h <= 0 {
			pw, ph, pErr := probeVideoDimensions(playURL)
			if pErr == nil {
				w, h = pw, ph
			}
		}
		sec := vmafSampleSeconds()
		if w > 0 && h > 0 {
			distorted := playURL
			cleanup := func() {}
			if strings.HasSuffix(strings.ToLower(strings.TrimSpace(playURL)), ".m3u8") || strings.EqualFold(strings.TrimSpace(playInfo.PlayInfoList[0].Format), "hls") {
				localPlaylist, cl, pErr := prepareHLSPlaylistForFFmpeg(config, playURL, 3600)
				if pErr == nil {
					distorted = localPlaylist
					cleanup = cl
				}
			}
			score, vErr := computeVMAF(inputPath, distorted, w, h, sec)
			cleanup()
			if vErr == nil {
				if sec > 0 {
					fmt.Printf("VMAF(%ds): %.2f\n", sec, score)
				} else {
					fmt.Printf("VMAF: %.2f\n", score)
				}
			} else {
				fmt.Printf("VMAF: -\n")
			}
		} else {
			fmt.Printf("VMAF: -\n")
		}
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

	inputDir := strings.TrimSpace(config.Video.InputPath)
	if inputDir == "" {
		inputDir = "./video"
	}
	st, err := os.Stat(inputDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("未找到输入路径: %s\n", inputDir)
			return
		}
		fmt.Fprintf(os.Stderr, "读取输入路径失败: %v\n", err)
		os.Exit(1)
	}

	var videoFiles []string
	if st.IsDir() {
		videoFiles, err = listVideoFiles(inputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取输入目录失败: %v\n", err)
			os.Exit(1)
		}
		if len(videoFiles) == 0 {
			if _, err := os.Stat("./hunli.MP4"); err == nil {
				videoFiles = []string{"./hunli.MP4"}
			} else if _, err := os.Stat("./hunli.mp4"); err == nil {
				videoFiles = []string{"./hunli.mp4"}
			} else {
				fmt.Printf("输入目录下未找到可处理的视频文件: %s\n", inputDir)
				return
			}
		}
	} else {
		videoFiles = []string{inputDir}
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

	results := make([]struct {
		LocalPath string
		PlayURL   string
	}, 0, len(videoFiles))

	for idx, inputPath := range videoFiles {
		fmt.Println("=" + strings.Repeat("=", 60))
		fmt.Printf("处理文件(%d/%d): %s\n", idx+1, len(videoFiles), inputPath)
		fmt.Println("=" + strings.Repeat("=", 60))
		fmt.Println()

		objectKey, fileSize, uploadDuration, err := uploadVideoToTOS(config, inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "上传失败: %v\n", err)
			continue
		}

		fmt.Println("\n注意: 如果遇到签名错误，建议使用火山引擎官方VOD SDK\n参考: https://www.volcengine.com/docs/4/3479")
		runId, err := startWorkflow(vodClient, config, objectKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n触发工作流失败: %v\n", err)
			fmt.Fprintf(os.Stderr, "\n提示: 如果遇到签名认证错误，请考虑:\n")
			fmt.Fprintf(os.Stderr, "1. 使用火山引擎官方VOD SDK\n")
			fmt.Fprintf(os.Stderr, "2. 检查AccessKey和SecretKey是否正确\n")
			fmt.Fprintf(os.Stderr, "3. 参考官方文档: https://www.volcengine.com/docs/4/3479\n\n")
			continue
		}

		vid, _, transcodeInfo, err := getWorkflowStatus(vodClient, runId)
		if err != nil {
			fmt.Fprintf(os.Stderr, "查询转码状态失败: %v\n", err)
			continue
		}

		vid, transcodeInfo, err = ensureHLSOutputAvailable(vodClient, config, objectKey, vid, transcodeInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "HLS产物校验失败: %v\n", err)
			continue
		}

		playUrl, playInfo, err := getPlayInfo(vodClient, config, vid, objectKey, transcodeInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取播放地址失败: %v\n", err)
			continue
		}

		displayTestResults(config, uploadDuration, fileSize, inputPath, playUrl, playInfo)

		fmt.Println("=" + strings.Repeat("=", 60))
		fmt.Println("本文件处理完成!")
		fmt.Println("=" + strings.Repeat("=", 60))
		fmt.Printf("播放地址: %s\n", playUrl)
		fmt.Println()

		results = append(results, struct {
			LocalPath string
			PlayURL   string
		}{LocalPath: inputPath, PlayURL: playUrl})
	}

	if len(results) > 0 {
		fmt.Println("=" + strings.Repeat("=", 60))
		fmt.Println("批量处理结果汇总")
		fmt.Println("=" + strings.Repeat("=", 60))
		for _, item := range results {
			fmt.Printf("%s\n%s\n\n", item.LocalPath, item.PlayURL)
		}
	}
}
