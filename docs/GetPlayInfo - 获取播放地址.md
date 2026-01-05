
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
登录
立即注册
视频点播
文档指南
请输入

文档首页

视频点播

服务端 API 参考

音视频播放

GetPlayInfo - 获取播放地址

复制全文
我的收藏
GetPlayInfo - 获取播放地址
调用 GetPlayInfo 传入 Vid 并指定封装格式、编码格式、清晰度等参数，获取所需的音视频播放地址。

使用说明
前提条件：
在视频点播中，音视频资源的发布状态分为发布和未发布。默认情况下，上传到视频点播的音视频的状态为未发布，无法获取播放地址进行播放。因此，在播放音视频之前，您必须将音视频的发布状态修改为已发布。具体请见修改音视频发布状态。
已添加并配置加速域名。
计费说明：添加加速域名后，从视频点播的播放地址直接下载或者播放视频将产生视频分发费用。具体请见分发加速计费。
播放降级逻辑说明：调用 GetPlayInfo 接口时，您可以通过指定封装格式、编码格式、清晰度等参数，从视频点播服务获取相应的播放地址。然而，有时您可能会发现接口返回的视频信息与预期不符，这可能是由于触发了播放降级逻辑。例如，视频点播服务仅转码生成了 H.264 视频，而业务请求的是 H.265 视频，或者业务请求加密视频时，仅将 FileType 设为 evideo，而 Format、Codec 等参数为空而使用了默认值，但视频点播服务无法找到对应规格的视频。在这些情况下，接口无法按照请求参数返回，播放服务会根据特定逻辑进行降级，选择备选的转码视频返回。
注意事项
本接口的单用户 QPS 限制为 2000 次/秒。超过限制，API 调用会被限流，这可能会影响您的业务，请合理调用。更多信息，请参见 QPS 限制。
请求说明
请求地址：https://vod.volcengineapi.com?Action=GetPlayInfo&Version=2020-08-01

调试
API Explorer
您可以通过 API Explorer 在线发起调用，无需关注签名生成过程，快速获取调用结果。
去调试
请求参数
下表仅列出该接口特有的请求参数和部分公共参数。更多信息请见公共请求参数。

参数
类型
是否必选
示例值
描述
Action
String
是
GetPlayInfo
接口名称。当前 API 的名称为 GetPlayInfo。
Version
String
是
2020-08-01
接口版本。当前 API 的版本为 2020-08-01。
Vid
String
是
v029c1g10003civ2i5mqib*******
视频 ID。将音视频上传至视频点播服务后，可通过媒体上传完成事件通知获取 Vid。
Format
String
否
mp4
封装格式。

当 FileType 为视频时，取值如下：
（默认）mp4
dash
hls
当 FileType 为音频时，取值如下：
（默认）m4a
mp4
mp3
dash
hls
ogg
Codec
String
否
H265
编码格式。

当 FileType 为视频时，取值如下：
（默认）H264
H265
H266
当 FileType 为音频时，取值如下：
（默认）aac（含 heaacv2）
mp3
opus
说明

如果客户端使用 Vid 和 PlayAuthToken 播放视频，建议您在应用服务端生成 PlayAuthToken 时，不要指定编码格式。客户端播放请求时，视频点播服务会根据策略自动灵活选择 H.264 或 H.265。
如需使用 H.266 编码，请提交工单联系技术支持开通。
Definition
String
否
1080p
视频流清晰度。仅当 FileType 为 video（非加密视频）和 evideo（加密视频）时生效。如未指定 Definition，接口将默认返回除自适应码流视频地址外的所有清晰度的视频播放地址。取值如下：

240p
360p
480p
540p
720p
1080p
2k
4k
od（原画转封装）
oe（画质增强）
auto（自适应码流 ABR）
说明

若将 Definition 设为 oe 获取画质增强视频，您需先通过视频点播媒体处理服务生成画质增强视频，完整介绍请见画质增强。若视频点播服务端没有 oe 视频，接口会返回空列表。若同时设置 Definition 和 HDRDefinition 为 oe，结果将取两者的并集。若均无 oe，则返回空列表。
若将 Definition 设为 auto 获取使用自适应码流 ABR 文件，您需先通过视频点播媒体处理服务生成自适应码流文件，完整介绍请见自适应码流转码和播放。仅当 Definition 为 auto 时，接口才会返回自适应码流的视频播放地址。如视频点播服务没有自适应码流文件，仅返回原始片源信息。您还需传入 Codec 和 Format，视频点播服务端才会下发对应的 auto 流。若未找到相应格式的 auto 流，则返回空列表。
FileType
String
否
video
流文件类型。取值如下：

video：（默认）非加密视频
audio：非加密音频
private_evideo: 私有加密视频（新版）
private_eaudio: 私有加密音频（新版）
standard_evideo: HLS 标准加密视频
standard_eaudio: HLS 标准加密音频
evideo：加密视频（旧版）
eaudio：加密音频（旧版）
LogoType
String
否
aa
水印贴片标签。即您在视频点播控制台创建水印贴片模板时设置的自定义水印贴片标签，详见水印贴片模板。
Ssl
String
否
1
是否返回 HTTPS 地址。取值如下：

1：是。
0：（默认）否。
说明

请确保您已配置 SSL 证书。

NeedThumbs
String
否
0
是否返回雪碧图信息。取值如下：

1：是。
0：（默认）否。
说明

请确保您已通过视频点播媒体处理服务生成雪碧图，完整介绍请见生成和使用雪碧图。

NeedBarrageMask
String
否
0
是否返回蒙版弹幕信息。取值如下：

1：是。
0：（默认）否。
说明

请确保您已通过视频点播媒体服务生成蒙版弹幕。

UnionInfo
String
否
87878***
播放端从浏览器或设备中取出的能够标识访问或设备唯一性的信息，用于播放火山引擎私有加密视频。详情请见火山引擎私有加密。
HDRDefinition
String
否
1080p
HDR 清晰度。默认为不获取 HDR 清晰度。取值如下：

240p
360p
480p
540p
720p
1080p
2k
4k
oe（画质增强）
说明

如 HDRDefinition 为 oe，但视频点播服务端没有 oe 视频，接口会返回空列表。
Definition 和 HDRDefinition 取值都为 oe 时，接口会返回并集。如果都没有 oe 视频，则返回空列表。
PlayScene
String
否
preview
播放场景。用于获取指定场景的音视频流。当前仅支持设为 preview，表示试看场景。详见视频试看。
DrmExpireTimestamp
String
否
1695037103
DRM 过期时间戳，用于加密音视频。Unix 时间戳，单位为秒。
Quality
String
否
higher
音频或视频质量。

当 FileType 为 audio 和 eaudio 时，如不传 Quality，默认返回全部音质的音频地址。取值如下：
medium：普通音质。
higher：高音质。
highest：音乐音质。
当 FileType 为 video 和 evideo 时，如不传 Quality，默认返回普通画质的视频地址。取值如下：
higher：更高画质
high：高画质
normal：普通画质
low：低画质
lower：更低画质
PlayConfig
String
否
{"PlayDomain":"vod.test_domain"}
自定义播放配置。为 JSON 字符串，支持指定播放域名。例如：{"PlayDomain":"vod.test_domain"}

说明

如果您设置了 PlayConfig 且其中的 PlayDomain 有效，点播服务会使用您传入的 PlayDomain 下发播放地址。
如果您未设置 PlayConfig 和其中的 PlayDomain，或者 PlayDomain 无效，视频点播服务会使用您在点播控制台分发加速设置 > 域名设置页面设置的默认域名下发播放地址。如果您没有设置默认域名，则会在启用的播放域名中随机分发。为防止视频点播服务返回的不是您想要的域名，建议您设置默认播放域名，详见域名设置。
NeedOriginal
String
否
0
是否返回片源信息。取值如下：

1：是。
0：（默认）否。
ForceExpire
String
否
60
强行指定本次请求的时间戳防盗链，单位为秒，取值范围为 [60,315360000]。

说明

该 ForceExpire 参数设置的过期时间优先级高于在视频点播控制台的域名管理中配置的时间戳防盗链。

GetAll
Boolean
否
false
是否返回全部流（包含片源和转码流）的播放信息。取值如下：

false：（默认）不开启。
true：开启。开启后将没有降级策略，返回 Vid 下所有流的信息。
注意

同时设置 NeedOriginal 和 GetAll 时，NeedOriginal 优先级高于 GetAll。具体如下：

NeedOriginal 不传，GetAll 设为 true，返回片源和转码流的播放信息。
NeedOriginal 设为 0，GetAll 设为 true，仅返回全部转码流的播放信息，不返回片源的播放信息。
DigitalWatermarkType
String
否
ABTraceStream
数字水印（暗水印）类型。当前仅支持设为 ABTraceStream（AB 流溯源水印）。您需要先通过自适应转码模板生成溯源水印 AB 流，然后在调用此接口时设置 DigitalWatermarkType 和 UserToken 参数从视频点播服务端获取溯源水印视频地址。完整使用流程请见暗水印。
UserToken
String
否
12345
溯源水印的用户凭证。当 DigitalWatermarkType 为 ABTraceStream 时需要设置此参数。视频点播会基于 UserToken 为该用户生成并下发唯一的溯源水印视频。UserToken 必须为数字，取值范围为 [0, 2^24)，即最多标记 16777216 个用户。请确保每个用户的 UserToken 具有唯一性。完整使用流程请见暗水印。
DrmKEK
String
否
DrmKEK
客户端播放器 SDK 生成的 Key Encryption Key，用于加密或封装视频加密密钥的密钥。此参数用于播放火山引擎私有加密视频。详细信息请参考火山引擎私有加密。
JSPlayer
String
否
1
标识请求是否来自于 Web 播放器 SDK。取值为 1 表明请求来自 Web 播放器 SDK。此参数用于播放火山引擎私有加密视频。详细信息请参考火山引擎私有加密。
返回参数
下表仅列出本接口特有的返回参数。更多信息请见公共返回参数。

参数
类型
示例值
描述
Vid
String
v030c9g1000***imqljht955iqlsog
视频 ID。
Status
Integer
10
视频状态，取值如下：

10：请求成功。
其他数值均表示视频无法播放。可能返回以下数值：
1000：视频未发布，不可播放。
1010：视频已被删除。
PosterUrl
String
https://img.example.com/1234/image.jpeg
封面图访问地址。
Duration
Float
0.1
文件时长，单位为秒。
FileType
String
video
流文件类型。取值如下：

evideo：加密视频流。
eaudio：加密音频流。
video：非加密视频流。
audio：普通音频流。
EnableAdaptive
Boolean
false
是否关键帧对齐。取值如下：

true：是。
false：否。
TotalCount
Integer
1
播放列表数量。
PlayInfoList
Object[]
-
播放信息列表。
FileId
String
v029c1***03civ2i5mqib
文件 ID。
Md5
String
398e352f8342aa29a6feee2a18e*****
文件 MD5 值。
FileType
String
video
文件类型，取值如下：

video
audio
Definition
String
1080p
视频分辨率。
Quality
String
normal
音频质量，取值如下：

medium：普通音质
higher：高音质
highest：音乐音质
Format
String
mp4
视频格式。
Codec
String
H265
编码类型。
LogoType
String
aa
水印贴片标签。即您在视频点播控制台创建水印贴片模板时配置的水印贴片标签，详见水印贴片模板。
MainPlayUrl
String
http://video.example.com/oIpB7fCZQ2ttb***
主播放地址。
BackupPlayUrl
String
http://video.example.com/oIpB7fCZQ2ttb***
备播放地址。
Bitrate
Integer
1381635
码率，单位为 bps。
Width
Integer
1920
视频宽度，单位为 px。
Height
Integer
1080
视频高度，单位为 px。
Size
Double
1653067
视频大小，单位为字节。
CheckInfo
String
a:v029c1g10003civ2i5mqib*******|b:0-983-ac368d50ea39b160********|c:0-983-****
劫持校验信息。此参数您无需关注。
IndexRange
String
974-1089
DASH segment_base 分片信息，用以描述 sidx（分段索引）的范围。
InitRange
String
0-973
DASH segment_base 分片信息，用以描述头信息的范围。
PlayAuth
String
****
加密过的密钥。用于播放私有加密视频。详细信息，请见：

火山引擎私有加密（旧版）
火山引擎私有加密（新版）
PlayAuthId
String
****
密钥 ID（Kid）。用于 Web 播放器 SDK 播放私有加密视频。详见播放私有加密视频（旧版）。
BarrageMaskOffset
String
100
蒙版弹幕偏移量。
MainUrlExpire
String
1690534212
主播放地址过期时间，Unix 时间戳。
BackupUrlExpire
String
1690534212
备播放地址过期时间，Unix 时间戳。
KeyFrameAlignment
String
-
使用的帧对齐转码版本
Volume
Object
-
音量均衡响度信息。
Peak
Double
0.1
音量峰值
Loudness
Double
0.1
音量响度
DrmType
String
private_encrypt_upgrade
加密类型：

private_encrypt：私有加密（旧版）。不会使用传入的 DrmKEK 进行加密。更多信息，请见火山引擎私有加密（旧版）。
private_encrypt_upgrade：私有加密（新版）。点播服务会使用请求参数中的 DrmKEK 对密钥进行加密。客户端需要使用保存的私钥进行解密。更多信息，请见火山引擎私有加密（新版）。
standard_encrypt: HLS 标准加密。不会使用传入的 DrmKEK 进行加密。更多信息，请见 HLS 标准加密。
PlayScene
String
preview
播放场景。
ThumbInfoList
Object[]
-
雪碧图信息列表。
CaptureNum
Integer
27
截取的小图总数。
StoreUrls
String[]
["http://vod-demo-cover.com/tos-example/622ceb09eb71_1639039860~tplv-vod-noop.image", "http://vod-demo-cover.com/tos-example/26d92b824_1639039860~tplv-vod-noop.image"]
雪碧图 URL 列表。
CellWidth
Integer
242
小图宽度，单位为 px。
CellHeight
Integer
136
小图高度，单位为 px。
ImgXLen
Integer
5
雪碧图每行包含的小图数量。
ImgYLen
Integer
5
雪碧图每列包含的小图数量。
Interval
Double
10
截图时间间隔，单位为秒
Format
String
jpg
图片格式，取值为 jpg。
SubtitleInfoList
Object[]
-
字幕信息列表。
Vid
String
v029c1g10003civ2i5mqib*******
视频 ID。
FileId
String
v029c1g10003civ2i5mqib*******
字幕文件 ID。
Language
String
eng-US
字幕语言。
LanguageId
Integer
1
字幕语言 ID。详见字幕语言。
Format
String
webvtt
字幕格式。
SubtitleId
String
123
字幕 ID。
Title
String
subtitle.vtt
字幕标题。
Tag
String
subtitle
字幕标签。
Status
String
enable
字幕状态。
Source
String
MU
字幕来源。
StoreUri
String
tos-vod-cn-**/191fe22a1c4a49************
字幕文件存储地址。
SubtitleUrl
String
http://example.com/191fe22a1c4a************
字幕文件访问地址。

说明

【暂不可用】该字段当前始终返回空值，请勿依赖此字段。

CreateTime
String
2019-10-12T07:20:50.52Z
创建时间。
Version
String
1
字幕版本。
BarrageMaskUrl
String
http://example.com/460224c***a9177b80b
蒙版弹幕 URL。
BarrageMaskInfo
Object
-
蒙版弹幕信息列表。
Version
String
v1
蒙版弹幕版本，取值如下：

V1
V2
BarrageMaskUrl
String
http://example.com/460224cfc86a9177b80b****
蒙版弹幕 URL。
FileId
String
v029c1g10003civ2i5mqib*******
蒙版弹幕文件 ID。
FileSize
Double
1
蒙版弹幕文件大小，单位为字节。
FileHash
String
c2e595ab2db511daf6fc************
蒙版弹幕文件哈希值。
UpdatedAt
String
1676017784
蒙版弹幕文件更新时间。
Bitrate
Integer
1
蒙版弹幕文件码率，单位为 bps。
HeadLen
Double
1
蒙版弹幕文件头部大小。
AdaptiveBitrateStreamingInfo
Object
-
自适应码流 ABR 主流信息。仅当 Definition 为 auto 时返回。

说明

可通过 PlayInfoList 获取自适应码流 ABR 子流信息。

AbrFormat
String
hls
自适应码流封装格式，取值为 hls 和 dash。
MainPlayUrl
String
http://video.example.com/oIpB7fCZQ2ttb***
主流主播放地址，仅当封装格式为 hls 时返回。封装格式为 dash 时，为空字符串。
BackupPlayUrl
String
http://video.example.com/oIpB7fCZQ2ttb***
主流备播放地址，仅当封装格式为 hls 时返回。封装格式为 dash 时，为空字符串。
请求示例
https://vod.volcengineapi.com?Action=GetPlayInfo&Version=2020-08-01&Vid=v029c1g10003civ2i5mqib*******&Format=mp4&Codec=H265&Definition=1080p&FileType=video&LogoType=&Base64=&Ssl=1&NeedThumbs=0&NeedBarrageMask=0&CdnType=&UnionInfo=&HDRDefinition=1080p&PlayScene=preview&DrmExpireTimestamp=1695037103&Quality=higher&PlayConfig={"PlayDomain":"vod.test_domain"}&NeedOriginal=0
返回示例
{
  "ResponseMetadata": {
    "RequestId": "202306041104200100100232280022D31",
    "Action": "GetPlayInfo",
    "Version": "2020-08-01",
    "Service": "vod",
    "Region": "cn-north-1"
  },
  "Result": {
    "Vid": "v029c1g10003civ2i5mqib*******",
    "Status": 1,
    "PosterUrl": "https://img.***.com/1234/8511abe****.jpeg",
    "Duration": 0.1,
    "FileType": "video",
    "EnableAdaptive": true,
    "TotalCount": 1,
    "PlayInfoList": [
      {
        "FileId": "v029c1g10003civ2i5mqib*******",
        "Md5": "398e352f8342aa29a6feee2a18e*****",
        "FileType": "video",
        "Definition": "1080p",
        "Quality": "normal",
        "Format": "mp4",
        "Codec": "H265",
        "LogoType": "default",
        "MainPlayUrl": "http://video.***.com/oIpB7fCZQ2ttbMe4Iklnvx********",
        "BackupPlayUrl": "http://video.***.com/oIpB7fCZQ2ttbMe4Iklnvx********",
        "Bitrate": 1381635,
        "Width": 1920,
        "Height": 1080,
        "Size": 1653067,
        "CheckInfo": "\"a:v029c1g10003civ2i5mqib*******|b:0-983-ac368d50ea39b160********|c:0-983-****\"",
        "IndexRange": "974-1089",
        "InitRange": "0-973",
        "PlayAuth": "****",
        "PlayAuthId": "****",
        "BarrageMaskOffset": "100",
        "Volume": {
          "Loudness": 0.1,
          "Peak": 0.1
        },
        "MainUrlExpire": "1690534212",
        "BackupUrlExpire": "1690534212"
      }
    ],
    "ThumbInfoList": [
      {
        "CaptureNum": 1,
        "StoreUrls": [
          "http://****.com/611d9a3c188150d69435************"
        ],
        "CellWidth": 1,
        "CellHeight": 1,
        "ImgXLen": 1,
        "ImgYLen": 1,
        "Interval": 0.1,
        "Format": "jpg"
      }
    ],
    "BarrageMaskUrl": "http://****.com/460224cfc86a9177b80b************",
    "SubtitleInfoList": [
      {
        "Vid": "v029c1g10003civ2i5mqib*******",
        "FileId": "v029c1g10003civ2i5mqib*******",
        "Language": "eng-US",
        "LanguageId": 1,
        "Format": "webvtt",
        "SubtitleId": "123",
        "Title": "subtitle.vtt",
        "Tag": "subtitle",
        "Status": "enable",
        "Source": "MU",
        "StoreUri": "tos-vod-cn-****/191fe22a1c4a49189a69************",
        "SubtitleUrl": "http://****.com/191fe22a1c4a49189a69************",
        "CreateTime": "2019-10-12T07:20:50.52Z",
        "Version": "1"
      }
    ],
    "BarrageMaskInfo": {
      "Version": "v1",
      "BarrageMaskUrl": "http://****.com/460224cfc86a9177b80b************",
      "FileId": "v029c1g10003civ2i5mqib*******",
      "FileSize": 1,
      "FileHash": "c2e595ab2db511daf6fc************",
      "UpdatedAt": "1676017784",
      "Bitrate": 1,
      "HeadLen": 1
    }
  }
}
错误码
下表列举了本接口特有的错误码。如需了解更多错误码，详见视频点播公共错误码。

状态码	错误码	错误信息	说明
400	InvalidParameter.InvalidVid	-	非法的 Vid。可能是因为请求参数内 Vid 为空或长度异常。
400	InvalidParameter.VidNotExist	-	Vid 不存在。
404	ResourceNotFound.NoAvailableDomain	-	未配置 CDN 域名。可排查是否开启点播域名调度，此为通过 GetPlayInfo 获取视频播放地址的必要操作。
服务端 SDK
视频点播为 OpenAPI 提供了配套的服务端 SDK，支持多种编程语言，帮助您实现快速开发。建议使用服务端 SDK 来调用 API。此 API 各语言调用的示例代码，请参考如下：

Java
Python
PHP
Go
Node.js
最近更新时间：2025.11.05 16:23:36
这个页面对您有帮助吗？
有用
无用
上一篇：
GetAdAuditResultByVid - 获取广告预审结果
CreateHlsDecryptionKey - 创建 HLS 标准加密密钥
下一篇
使用说明
注意事项
请求说明
调试
请求参数
返回参数
请求示例
返回示例
错误码
服务端 SDK

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
