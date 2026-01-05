
文档中心
火山方舟大模型服务平台
豆包语音
扣子
API网关
云服务器
火山方舟大模型服务平台
豆包语音
扣子
API网关
云服务器
文档
备案
控制台
z
zhaoweibo.0820 / eps_yxd_group
账号管理
账号ID : 2108323502
联邦登陆
企业认证
费用中心
可用余额¥ 0.00
充值汇款
账户总览
账单详情
费用分析
发票管理
权限与安全
安全设置
访问控制
操作审计
API 访问密钥
工具与其他
公测申请
资源管理
配额中心
伙伴控制台
待办事项
待支付
0
待续费
0
待处理工单
0
未读消息
0
视频点播
文档指南
请输入

文档首页

视频点播

服务端 SDK

V1.0

Go SDK

音视频播放

复制全文
我的收藏
音视频播放
本文为您提供了服务端 Go SDK 的媒资播放模块的接口调用示例。

前提条件
调用接口前，请先完成集成 Go SDK。

调用示例

签发临时播放 token
对于存储在视频点播中的音视频，您可以使用客户端播放器 SDK 通过临时播放 Token 自动获取播放地址进行播放。具体播放流程和临时播放 Token 说明，请见通过临时播放 Token 播放。为方便您的使用，视频点播服务端 SDK 对临时播放 Token 的签发进行了封装。您可调用生成方法通过 AK/SK 在本地签出临时播放 Token，不依赖外网。若希望同时生成多个临时播放 Token，可循环调用生成方法。

说明

接口请求参数和返回参数说明详见获取播放地址。

package vod

import (
        "encoding/json"
        "fmt"
        "github.com/volcengine/volc-sdk-golang/service/vod/upload/model"
        "testing"
        "time"

        "github.com/volcengine/volc-sdk-golang/service/vod"
        "github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

func TestVod_GetPlayAuthToken(t *testing.T) {
        vid := "your vid"
        tokenExpireTime := 600 // Token Expire Duration（s）
        // Create a VOD instance in the specified region.
        // instance := vod.NewInstanceWithRegion("cn-north-1")
        instance := vod.NewInstance()

        // Configure your Access Key ID (AK) and Secret Access Key (SK) in the environment variables or in the local ~/.volc/config file. For detailed instructions, see  https://www.volcengine.com/docs/4/65655.
        // The SDK will automatically fetch the AK and SK from the environment variables or the ~/.volc/config file as needed.
        // During testing, you may use the following code snippet. However, do not store the AK and SK directly in your project code to prevent potential leakage and safeguard the security of all resources associated with your account.
        // instance.SetCredential(base.Credentials{
        // AccessKeyID:     "your ak",
        // SecretAccessKey: "your sk",
        //})

        query := &request.VodGetPlayInfoRequest{
                Vid:                  vid,
                Format:               "your Format",
                Codec:                "your Codec",
                Definition:           "your Definition",
                FileType:             "your FileType",
                LogoType:             "your LogoType",
                Ssl:                  "your Ssl",
                NeedThumbs:           "your NeedThumbs",
                NeedBarrageMask:      "your NeedBarrageMask",
                UnionInfo:            "your UnionInfo",
                HDRDefinition:        "your HDRDefinition",
                PlayScene:            "your PlayScene",
                DrmExpireTimestamp:   "your DrmExpireTimestamp",
                Quality:              "your Quality",
                PlayConfig:           "your PlayConfig",
                NeedOriginal:         "your NeedOriginal",
                ForceExpire:          "your ForceExpire",
                GetAll:               false,
                DigitalWatermarkType: "your DigitalWatermarkType",
                UserToken:            "your UserToken",
                DrmKEK:               "your DrmKEK",
                JSPlayer:             "your JSPlayer",
        }
        newToken, _ := instance.GetPlayAuthToken(query, tokenExpireTime)
        fmt.Println(newToken)
}

获取播放地址
接口请求参数和返回参数详见 OpenAPI：获取播放地址。

package vod

import (
        "encoding/json"
        "fmt"
        "github.com/volcengine/volc-sdk-golang/service/vod/upload/model"
        "testing"
        "time"

        "github.com/volcengine/volc-sdk-golang/service/vod"
        "github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

func Test_GetPlayInfo(t *testing.T) {
        // Create a VOD instance in the specified region.
        // instance := vod.NewInstanceWithRegion("cn-north-1")
        instance := vod.NewInstance()
        
        // Configure your Access Key ID (AK) and Secret Access Key (SK) in the environment variables or in the local ~/.volc/config file. For detailed instructions, see  https://www.volcengine.com/docs/4/65655.
        // The SDK will automatically fetch the AK and SK from the environment variables or the ~/.volc/config file as needed.
        // During testing, you may use the following code snippet. However, do not store the AK and SK directly in your project code to prevent potential leakage and safeguard the security of all resources associated with your account.
        // instance.SetCredential(base.Credentials{
        // AccessKeyID:     "your ak",
        // SecretAccessKey: "your sk",
        //})

        query := &request.VodGetPlayInfoRequest{
                Vid:                  "your Vid",
                Format:               "your Format",
                Codec:                "your Codec",
                Definition:           "your Definition",
                FileType:             "your FileType",
                LogoType:             "your LogoType",
                Ssl:                  "your Ssl",
                NeedThumbs:           "your NeedThumbs",
                NeedBarrageMask:      "your NeedBarrageMask",
                UnionInfo:            "your UnionInfo",
                HDRDefinition:        "your HDRDefinition",
                PlayScene:            "your PlayScene",
                DrmExpireTimestamp:   "your DrmExpireTimestamp",
                Quality:              "your Quality",
                PlayConfig:           "your PlayConfig",
                NeedOriginal:         "your NeedOriginal",
                ForceExpire:          "your ForceExpire",
                GetAll:               false,
                DigitalWatermarkType: "your DigitalWatermarkType",
                UserToken:            "your UserToken",
                DrmKEK:               "your DrmKEK",
                JSPlayer:             "your JSPlayer",
        }

        resp, status, err := instance.GetPlayInfo(query)
        fmt.Println(status)
        fmt.Println(err)
        fmt.Println(resp.String())
}

签发私有加密 Token
私有加密 Token 用于 Web 播放器 SDK 播放私有加密视频，详情请见以下文档：

火山引擎私有加密方案
播放私有加密视频（新版）
私有加密 Token 可由 App/Web Server 持有的 AK/SK 在本地签出，不依赖外网。若希望同时生成多个私有加密 Token，您可以循环调用生成方法。

package vod

import (
        "encoding/json"
        "fmt"
        "github.com/volcengine/volc-sdk-golang/service/vod/upload/model"
        "testing"
        "time"

        "github.com/volcengine/volc-sdk-golang/service/vod"
        "github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

func TestVod_GetPrivateDrmAuthToken(t *testing.T) {
        // Create a VOD instance in the specified region.
        // instance := vod.NewInstanceWithRegion("cn-north-1")
        instance := vod.NewInstance()

        // Configure your Access Key ID (AK) and Secret Access Key (SK) in the environment variables or in the local ~/.volc/config file. For detailed instructions, see  https://www.volcengine.com/docs/4/65655.
        // The SDK will automatically fetch the AK and SK from the environment variables or the ~/.volc/config file as needed.
        // During testing, you may use the following code snippet. However, do not store the AK and SK directly in your project code to prevent potential leakage and safeguard the security of all resources associated with your account.
        // instance.SetCredential(base.Credentials{
        // AccessKeyID:     "your ak",
        // SecretAccessKey: "your sk",
        //})
        
        // Web 播放器 SDK 内部携带 playAuthId、vid、unionInfo 向应用服务端请求获取私有加密 Token，详细示例代码和参数说明请见 [Web 端播放私有加密视频实现方式部分](https://www.volcengine.com/docs/4/68698#%E5%AE%9E%E7%8E%B0%E6%96%B9%E5%BC%8F-2)
        query := &request.VodGetPrivateDrmPlayAuthRequest{
                Vid:         "your vid",
                // 需指定 DrmType 为 webdevice
                DrmType:     "your drmType",
                PlayAuthIds: "your playAuthIds",
                UnionInfo:   "your unionInfo",
        }
        tokenExpireTime := 6000000 // Token Expire Duration（s）
        newToken, _ := instance.GetPrivateDrmAuthToken(query, tokenExpireTime)
        fmt.Println(newToken)
}

签发 HLS 标准加密 Token
由 App/Web Server 持有的 AK/SK 在本地签出，不依赖外网。若希望同时生成多个 HLS 标准加密 Token，您可以循环调用生成方法。HLS 标准加密 Token 用于 Web 客户端播放 HLS 加密音视频，详见 HLS 标准加密（旧版）。

package vod

import (
        "encoding/json"
        "fmt"
        "github.com/volcengine/volc-sdk-golang/service/vod/upload/model"
        "testing"
        "time"

        "github.com/volcengine/volc-sdk-golang/service/vod"
        "github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

func TestVod_GetSha1HlsDrmAuthToken(t *testing.T) {
        // Create a VOD instance in the specified region.
        // instance := vod.NewInstanceWithRegion("cn-north-1")
        instance := vod.NewInstance()
        
        // Configure your Access Key ID (AK) and Secret Access Key (SK) in the environment variables or in the local ~/.volc/config file. For detailed instructions, see  https://www.volcengine.com/docs/4/65655.
        // The SDK will automatically fetch the AK and SK from the environment variables or the ~/.volc/config file as needed.
        // During testing, you may use the following code snippet. However, do not store the AK and SK directly in your project code to prevent potential leakage and safeguard the security of all resources associated with your account.
        // instance.SetCredential(base.Credentials{
        // AccessKeyID:     "your ak",
        // SecretAccessKey: "your sk",
        //})
        
        expireDuration := int64(6000000) //change to your expire duration (s), no default duration
        token, _ := instance.CreateSha1HlsDrmAuthToken(expireDuration)
        fmt.Println(token)
}

创建 HLS 标准加密密钥
接口请求参数和返回参数说明详见 CreateHlsDecryptionKey - 创建 HLS 标准加密密钥。

package vod

import (
        "encoding/json"
        "fmt"
        "github.com/volcengine/volc-sdk-golang/service/vod/upload/model"
        "testing"
        "time"

        "github.com/volcengine/volc-sdk-golang/service/vod"
        "github.com/volcengine/volc-sdk-golang/service/vod/models/request"
)

func Test_CreateHlsDecryptionKey(t *testing.T) {
        // Create a VOD instance in the specified region.
        // instance := vod.NewInstanceWithRegion("cn-north-1")
        instance := vod.NewInstance()

        // Configure your Access Key ID (AK) and Secret Access Key (SK) in the environment variables or in the local ~/.volc/config file. For detailed instructions, see  https://www.volcengine.com/docs/4/65655.
        // The SDK will automatically fetch the AK and SK from the environment variables or the ~/.volc/config file as needed.
        // During testing, you may use the following code snippet. However, do not store the AK and SK directly in your project code to prevent potential leakage and safeguard the security of all resources associated with your account.
        // instance.SetCredential(base.Credentials{
        // AccessKeyID:     "your ak",
        // SecretAccessKey: "your sk",
        //})

        query := &request.VodCreateHlsDecryptionKeyRequest{
                SpaceName: "your SpaceName",
        }

        resp, status, err := instance.CreateHlsDecryptionKey(query)
        fmt.Println(status)
        fmt.Println(err)
        fmt.Println(resp.String())
}
最近更新时间：2025.12.19 11:24:35
这个页面对您有帮助吗？
有用
无用
上一篇：
媒体处理
分发加速
下一篇
前提条件
调用示例
签发临时播放 token
获取播放地址
签发私有加密 Token
签发 HLS 标准加密 Token
创建 HLS 标准加密密钥

鼠标选中内容，快速反馈问题
选中存在疑惑的内容，即可快速反馈问题，我们将会跟进处理
不再提示
好的，知道了

全天候售后服务
7x24小时专业工程师品质服务

极速服务应答
秒级应答为业务保驾护航

客户价值为先
从服务价值到创造客户价值

全方位安全保障
打造一朵“透明可信”的云
logo
关于我们
为什么选火山
文档中心
联系我们
人才招聘
云信任中心
友情链接
产品
云服务器
GPU云服务器
机器学习平台
客户数据平台 VeCDP
飞连
视频直播
全部产品
解决方案
汽车行业
金融行业
文娱行业
医疗健康行业
传媒行业
智慧文旅
大消费
服务与支持
备案服务
服务咨询
建议与反馈
廉洁舞弊举报
举报平台
联系我们
业务咨询：service@volcengine.com
市场合作：marketing@volcengine.com
电话：400-850-0030
地址：北京市海淀区北三环西路甲18号院大钟寺广场1号楼

微信公众号

抖音号

视频号
© 北京火山引擎科技有限公司 2025 版权所有
代理域名注册服务机构：新网数码 商中在线
服务条款
隐私政策
更多协议

京公网安备11010802032137号
京ICP备20018813号-3
营业执照
增值电信业务经营许可证京B2-20202418，A2.B1.B2-20202637
网络文化经营许可证：京网文（2023）4872-140号
