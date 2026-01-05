# 大疆云相册 - 视频上传转码Demo

这是一个基于火山引擎TOS（对象存储）和VOD（视频点播）服务的Golang Demo，实现了视频上传、转码和播放的完整流程。

## 功能特性

1. **视频上传到TOS桶** - 将本地视频文件上传到火山引擎对象存储
2. **视频转码** - 将原始视频转码为720P格式
3. **获取播放地址** - 获取转码后视频的播放链接
4. **性能测试** - 测试上传速度、压缩比等指标

## 前置条件

1. 已创建对象存储桶 `dji-vod-poc`（位于华北2-北京）
2. 已创建视频点播空间 `space-dji`
3. 已完成点播空间挂载对象存储桶
4. 已在控制台创建720P转码工作流模板

## 环境要求

- Go 1.21 或更高版本
- 火山引擎账号及AccessKey/SecretKey

## 安装步骤

### 1. 克隆或下载项目

```bash
cd /Users/bytedance/Documents/实验/大疆云相册
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置参数

**方式一：使用.env文件（推荐）**

复制 `.env_example` 文件为 `.env` 并填写真实值：

```bash
cp .env_example .env
```

编辑 `.env` 文件：

```bash
# 火山引擎统一配置（TOS和VOD使用相同的AK/SK）
ACCESS_KEY=your-access-key-here
SECRET_KEY=your-secret-key-here

# TOS配置（使用统一AK/SK）
TOS_ACCESS_KEY=your-access-key-here
TOS_SECRET_KEY=your-secret-key-here

# VOD配置（使用统一AK/SK）
VOD_ACCESS_KEY=your-access-key-here
VOD_SECRET_KEY=your-secret-key-here

# 工作流ID
VOD_WORKFLOW_ID=your-workflow-id-here
```

**注意：** 如果TOS和VOD使用相同的AK/SK（通常情况），只需设置 `ACCESS_KEY` 和 `SECRET_KEY`，程序会自动应用到TOS和VOD配置。如果它们使用不同的AK/SK，可以分别设置 `TOS_ACCESS_KEY`/`TOS_SECRET_KEY` 和 `VOD_ACCESS_KEY`/`VOD_SECRET_KEY`。

**方式二：使用config.yaml文件**

编辑 `config.yaml` 文件，填写以下配置：

```yaml
# TOS配置
tos:
  access_key: ""  # 从.env文件或环境变量TOS_ACCESS_KEY获取
  secret_key: ""  # 从.env文件或环境变量TOS_SECRET_KEY获取
  endpoint: "https://tos-cn-beijing.volces.com"
  region: "cn-beijing"
  bucket_name: "dji-vod-poc"

# VOD配置
vod:
  access_key: ""  # 从.env文件或环境变量VOD_ACCESS_KEY获取
  secret_key: ""  # 从.env文件或环境变量VOD_SECRET_KEY获取
  region: "cn-north-1"
  space_name: "space-dji"
  workflow_id: "d4dafb5eb8c0477eac792f331f875fdc"  # 或从.env文件VOD_WORKFLOW_ID获取
  api_endpoint: "https://vod.volcengineapi.com"

# 视频配置
video:
  input_path: "./video/原始婚礼片段.mp4"
  output_key_prefix: "videos/"
```

**方式三：使用系统环境变量**

```bash
export TOS_ACCESS_KEY="your-tos-access-key"
export TOS_SECRET_KEY="your-tos-secret-key"
export VOD_ACCESS_KEY="your-vod-access-key"
export VOD_SECRET_KEY="your-vod-secret-key"
export VOD_WORKFLOW_ID="your-workflow-id"
```

**配置优先级：** `.env` 文件 > 系统环境变量 > `config.yaml` 文件

## 使用方法

### 运行Demo

```bash
go run main.go vod_client.go
```

或者编译后运行：

```bash
go build -o demo main.go vod_client.go
./demo
```

### 执行流程

Demo将按以下步骤执行：

1. **上传视频到TOS桶**
   - 读取本地视频文件
   - 上传到指定的TOS桶
   - 显示上传速度和耗时

2. **触发视频转码工作流**
   - 调用VOD API触发转码任务
   - 获取工作流任务ID (RunId)

3. **查询转码状态**
   - 轮询查询转码任务状态
   - 等待转码完成
   - 获取视频ID (Vid)

4. **获取播放地址**
   - 调用GetPlayInfo API获取720P格式的播放地址
   - 显示视频详细信息（分辨率、码率、文件大小等）

5. **显示测试结果**
   - 上传速度
   - 压缩比
   - 转码后视频信息

## 工作流配置说明

在运行Demo之前，需要在火山引擎视频点播控制台创建720P转码工作流：

1. 登录火山引擎视频点播控制台
2. 进入空间 `space-dji`
3. 创建视频转码模板（720P）
4. 创建工作流，关联转码模板
5. 获取工作流ID，填写到 `config.yaml` 的 `vod.workflow_id`

## 输出示例

```
大疆云相册 - 视频上传转码Demo

============================================================
步骤1: 上传视频到TOS桶
============================================================
文件路径: ./video/原始婚礼片段.mp4
文件大小: 149.00 MB
对象Key: videos/原始婚礼片段.mp4
上传成功!
Request ID: xxxxx
上传耗时: 15.23 秒
上传速度: 9.78 MB/s

============================================================
步骤2: 触发视频转码工作流
============================================================
空间名称: space-dji
工作流ID: xxxxx
文件路径: videos/原始婚礼片段.mp4
存储桶: dji-vod-poc
正在触发工作流...
工作流任务ID (RunId): xxxxx

============================================================
步骤3: 查询工作流执行状态
============================================================
第 1 次查询状态...
当前状态: Running
转码进行中，等待 10s 后重试...
...
转码完成!
视频ID (Vid): v0c931g7007acxxxxx

============================================================
步骤4: 获取播放地址
============================================================
播放地址获取成功!
主播放地址: http://example.com/play/xxxxx.mp4
视频格式: mp4
编码格式: H264
清晰度: 720p
分辨率: 1280x720
码率: 2000000 bps
文件大小: 45.23 MB

============================================================
测试结果汇总
============================================================
上传速度: 9.78 MB/s
上传耗时: 15.23 秒
原始文件大小: 149.00 MB
转码后文件大小: 45.23 MB
压缩比: 30.36%
转码后分辨率: 1280x720
转码后码率: 2000000 bps
```

## 测试功能

Demo支持以下测试指标：

- **上传速度测试** - 自动计算上传速度（MB/s）
- **压缩比测试** - 对比原始视频和转码后视频的文件大小
- **播放器时延** - 使用提供的HTML播放器测试播放时延

### 使用HTML播放器测试

Demo执行完成后会输出播放地址，您可以使用项目中的 `player.html` 文件来测试播放：

1. 在浏览器中打开 `player.html`
2. 将播放地址粘贴到输入框中
3. 点击"加载视频"
4. 点击"测试播放时延"按钮测试播放延迟

或者直接在浏览器中访问：
```
file:///path/to/player.html?url=<播放地址>
```

## 注意事项

1. **敏感信息保护** - `.env` 文件包含敏感信息，已添加到 `.gitignore`，不会被提交到版本控制
2. **API签名** - 当前实现使用自定义签名逻辑，如果遇到签名错误，建议使用火山引擎官方SDK
3. **工作流ID** - 当前已配置工作流ID `d4dafb5eb8c0477eac792f331f875fdc`，如需修改可在 `.env` 文件中设置 `VOD_WORKFLOW_ID`
4. **转码时间** - 转码时间取决于视频大小和复杂度，Demo会每10秒轮询一次状态
5. **播放地址** - 获取播放地址需要确保已配置加速域名并开启点播调度
6. **费用** - 视频上传、转码和播放都会产生费用，请注意控制台费用

## 故障排查

### 1. 上传失败

- 检查TOS AccessKey和SecretKey是否正确
- 检查存储桶名称是否正确
- 检查文件路径是否存在

### 2. 转码失败 - API签名错误

如果遇到 `InvalidCredential` 错误，说明API签名有问题。当前Demo使用了自定义签名实现，可能不完全符合火山引擎的签名规范。

**解决方案：**

1. **使用火山引擎官方SDK（推荐）**
   ```bash
   go get github.com/volcengine/volc-sdk-golang/service/vod
   ```
   然后使用官方SDK替换当前的VOD客户端实现。

2. **检查Secret Key格式**
   - 确保Secret Key是base64编码格式
   - 当前代码已尝试自动解码base64格式的Secret Key

3. **参考官方文档**
   - 火山引擎API签名规范：https://www.volcengine.com/docs/4/3479
   - 确保签名算法完全符合规范

### 3. 获取播放地址失败

- 检查视频是否已转码完成
- 检查是否配置了加速域名
- 检查视频状态是否为已发布

## 依赖包

- `github.com/volcengine/ve-tos-golang-sdk/v2` - TOS SDK
- `gopkg.in/yaml.v3` - YAML配置文件解析

## 参考文档

- [对象存储普通上传](docs/对象存储普通上传.md)
- [音视频转码](docs/音视频转码.md)
- [GetPlayInfo - 获取播放地址](docs/GetPlayInfo%20-%20获取播放地址.md)
- [GetWorkflowExecution - 获取工作流运行状态](docs/GetWorkflowExecution%20-%20获取工作流运行状态.md)
- [点播添加对象存储](docs/点播添加对象存储.md)

## 许可证

本项目仅供演示使用。

