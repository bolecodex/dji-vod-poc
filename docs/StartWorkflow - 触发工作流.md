
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

服务端 API 参考

媒体处理

StartWorkflow - 触发工作流

复制全文
我的收藏
StartWorkflow - 触发工作流
调用 StartWorkflow 接口指定工作流 ID 触发媒体处理任务，以对指定片源进行媒体处理。

使用说明
使用该接口前，请确保您已充分了解视频点播的收费方式和价格。媒体处理为付费功能，具体请见媒体处理计费。
使用该接口前，请确保您已了解工作流机制。
成功提交后，系统将生成异步执行的媒体处理任务。您可通过以下方式获取媒体处理结果：
配置工作流执行完成事件通知。
调用 GetWorkflowExecutionResult 获取工作流执行结果。
注意事项
本接口的参数设置优先级高于您在视频点播控制台媒体处理模板和工作流中的参数设置。
本接口的单用户 QPS 限制为 50 次/秒。超过限制，API 调用会被限流，这可能会影响您的业务，请合理调用。更多信息，请参见 QPS 限制。
请求说明
请求地址：https://vod.volcengineapi.com?Action=StartWorkflow&Version=2020-08-01

调试
API Explorer
您可以通过 API Explorer 在线发起调用，无需关注签名生成过程，快速获取调用结果。
去调试
请求参数
下表仅列出该接口特有的请求参数和部分公共参数。更多信息请见公共请求参数。

参数	类型	是否必选	示例值	描述
Action	String	是	StartWorkflow	接口名称。当前 API 的名称为 StartWorkflow。
Version	String	是	2020-08-01	接口版本。当前 API 的版本为 2020-08-01。
Vid

String

否

v023bbg10001***2n6qiboeb9dn73mg

待处理视频的 Vid。您需要在调用本接口前，通过任一方式将视频上传至点播空间，并获取其 Vid。

注意

您必须设置 Vid 或 DirectUrl 参数之一，但不能同时设置这两个参数。

DirectUrl

Object of DirectUrl

否

-

通过 DirectUrl 模式指定待处理视频的信息。您需要在调用本接口前，通过任一方式将视频上传至点播空间，并获取其相关信息。

注意

您必须设置 Vid 或 DirectUrl 参数之一，但不能同时设置这两个参数。
如果是 POST 请求，该参数类型为 String。示例：
"{\"FileName\":\"example.mp4\"}"
TemplateId	String	是	63d92a1***b1795	工作流 ID。您可在视频点播控制台创建工作流并获取工作流 ID，详见工作流。
Input

Object of WorkflowParams

否

-

动态参数。默认情况下，视频点播根据工作流及其关联的媒体处理模板中的预设参数执行媒体处理任务。如果您需要在触发工作流时动态设置处理参数，例如设置输出文件路径或文字水印内容，您可以传入动态参数。动态参数将覆盖指定模板中相应参数的内容。

说明

如果是 POST 请求，该参数类型为 String。示例：
"{\"OverrideParams\":{\"Logo\":[{\"Vars\":{\"input_text\":\"文字水印测试\"}}]}}"

Priority	Integer	否	0	任务优先级。默认值为 0。取值范围为 [-5,5]。数字越大，优先级越高。
CallbackArgs	String	否	YourCallbackArgs	自定义字段，将在工作流执行完成事件中透传返回。字段长度最大为 512 字节。
EnableLowPriority

Boolean

否

false

是否开启闲时转码。取值如下：

true：开启。
false：关闭。
说明

对于闲时转码功能的介绍和使用场景，请见闲时转码。

ClientToken	String	否	QqS3o5L4BLFCDxGZ&b	用户请求凭证，用于区分不同请求。大小写敏感，不超过 64 个 ASCII 码可打印字符。默认情况下，视频点播转码服务在接收到触发工作流请求后，会执行幂等行为。这意味着，如果判定当前请求与之前发送的请求相同，则不会重复执行该请求。关于幂等行为、目的和条件，请参考幂等行为说明。若您希望跳过幂等行为，例如修改了某一工作流所绑定的媒体处理模板中的配置，想要对同一个 Vid 重新执行相同的工作流，则可以设置此参数。
DirectUrl
参数	类型	是否必选	示例值	描述
SpaceName

String

否

test

待处理文件所在点播空间的名称。您需要在调用本接口前创建空间，并在此处传入空间名称。

注意

若您触发的工作流是系统内置工作流，则 SpaceName 必填。

FileName	String	是	example.mp4	待处理文件的 FileName。您需要在调用本接口前，通过任一方式将视频上传至点播空间，并获取其 FileName。
BucketName	String	否	tos-cn-v-cd3a11	空间所绑定的存储桶名称。可调用 ListSpace 获取空间所绑定的存储桶名称。
WorkflowParams
参数	类型	是否必选	示例值	描述
OverrideParams	Object of OverrideParams	否	-	覆盖参数，用于覆盖媒体处理模板中的配置。
OverrideParams
参数	类型	是否必选	示例值	描述
Logo	Array of LogoOverride	否	[{"TemplateId":"1091058d***6b8f408c352ee1c52","Vars":{"keyword":"hello"}}]	水印覆盖参数。
Snapshot	Array of SnapshotOverride	否	[{"TemplateId": ["94c8d47c4c39440b8***e0ce7a96776"],"FileName": "{{fileTitle}}_{{templateId}}"}]	截图覆盖参数。
TranscodeAudio	Array of TranscodeAudioOverride	否	[{"TemplateId":["1091058d***6b8f408c352ee1c52"],"Clip":{"StartTime":0,"EndTime":20000}}]	音频转码覆盖参数。
TranscodeVideo	Array of TranscodeVideoOverride	否	[{"TemplateId":["1091058d***6b8f408c352ee1c52"],"Clip":{"StartTime":0,"EndTime":20000}}]	视频转码覆盖参数。
LogoOverride
参数	类型	是否必选	示例值	描述
Vars

JSON Map

否

{"keyword":"hello"}

水印覆盖内容。格式为 "key":"value"。

"key"：String 类型。该值为您在水印贴片模板中配置的自定义变量 key。具体步骤，请见动态替换文字水印内容。
"value"：String 类型。该值为自定义文字水印内容。
TemplateId	String	是	1091058d0**6b8f408c352ee1c52	水印贴片模板 ID。支持设为 All，表示对所有水印贴片模板生效。
SnapshotOverride
参数	类型	是否必选	示例值	描述
FileName

String

否

{{fileTitle}}

动态设置截图文件路径。文件路径可由固定字符串和变量组成，详见文件路径组成说明。假设你对文件名称为 vodtest 的片源触发工作流 a378f66a5bc2495eb55fc14ac109625f，并将 FileName 设为 {{fileTitle}}_{{templateId}}，则此次截图生成的文件路径为 vodtest_a378f66a5bc2495eb55fc14ac109625f。

注意

封装格式为 DASH 时，该参数不生效。

FileIndex	String	否	{{fileTitle}}	动态设置采样截图 index 文件路径。仅当截图类型为采样截图时生效。文件路径可由固定字符串和变量组成，详见文件路径组成说明。假设你对文件名称为 vodtest 的片源触发工作流 a378f66a5bc2495eb55fc14ac109625f，并将 FileIndex 设为 {{fileTitle}}_{{templateId}}，则此次截图生成的文件路径为 vodtest_a378f66a5bc2495eb55fc14ac109625f。
OffsetTime	Integer	否	100	截图时间间隔。单位为毫秒。仅当截图类型为静态图、动图、反复循环动图时生效。
TemplateId	Array of String	是	["1091058d0**6b8f408c352ee1c52"]	截图模板 ID 列表。支持取值为 All，表示不区分模板，只要为该类型就生效。
SampleOffsets	Array of Float	否	[1.0,2.0]	采样截图时间点。单位为秒。针对 Sample 类型截图模板有效。
OffsetTimeList	Array of Integer	否	[10,20]	多动图截图时间。单位为毫秒。针对 Dynpost 类型截图模板有效。
Width

Integer

否

480

采样截图宽度，单位为 px。取值范围为 [0, 4096]。

不传 Width 时，按照采样截图模板中配置的宽度进行截图。
Width 为 0 时，根据 Height 的值按照片源宽高比进行缩放。
Width 和 Height 均为 0 时，按照片源尺寸截图。
注意

当前仅支持采样截图，且仅在采样截图模板中图片尺寸设为固定尺寸时生效。

Height

Integer

否

720

采样截图高度，单位为 px。取值范围为 [0, 4096]。

不传 Height 时，按照采样截图模板中配置的高度进行截图。
Height 为 0 时，根据 Width 的值按照片源宽高比进行缩放。
Width 和 Height 均为 0 时，按照片源尺寸截图。
注意

当前仅支持采样截图，且仅在采样截图模板中图片尺寸设为固定尺寸时生效。

TranscodeAudioOverride
参数	类型	是否必选	示例值	描述
Clip	Object of Clip	否	{"StartTime":0,"EndTime":20000}	裁剪参数。
FileName

String

否

{{fileTitle}}

动态设置转码输出视频的文件路径。文件路径可由固定字符串和变量组成，详见文件路径组成说明。假设你对文件名称为 vodtest 的片源触发工作流 a378f66a5bc2495eb55fc14ac109625f，并将 FileName 设为 {{fileTitle}}_{{templateId}}，则此次转码生成的文件路径为 vodtest_a378f66a5bc2495eb55fc14ac109625f。

注意

封装格式为 DASH 时，该参数不生效。

TemplateId	Array of String	是	["1091058d0**6b8f408c352ee1c52"]	音频转码模板 ID 列表。支持 All 取值，此时不区分模板，只要为该类型就生效。
TranscodeVideoOverride
参数	类型	是否必选	示例值	描述
Clip	Object of Clip	否	{"StartTime":0,"EndTime":20000}	裁剪参数。
FileName

String

否

{{fileName}}

动态设置转码输出视频的文件路径。文件路径可由固定字符串和变量组成，详见文件路径组成说明。假设你对文件名称为 vodtest 的片源触发工作流 a378f66a5bc2495eb55fc14ac109625f，并将 FileName 设为 {{fileTitle}}_{{templateId}}，则此次转码生成的文件路径为 vodtest_a378f66a5bc2495eb55fc14ac109625f。

注意

封装格式为 DASH 时，该参数不生效。

TemplateId	Array of String	是	["1091058****6b8f408c352ee1c52"]	视频处理模板或极智超清模板 ID 列表。支持取值为 All，表示不区分模板，只要为该类型就生效。
LogoTemplateId	String	否	1091058****6b8f408c352ee1c52	水印贴片模板 ID。该参数仅当 TemplateId 所对应的模板是视频转码或极智超清类型时生效。
Clip
参数	类型	是否必选	示例值	描述
EndTime	Integer	否	0	裁剪结束时间，单位为毫秒。
StartTime	Integer	否	2000	裁剪开始时间，单位为毫秒。
返回参数
下表仅列出本接口特有的返回参数。更多信息请见公共返回参数。

参数	类型	示例值	描述
RunId	String	lb:0f557116******059cedf92f1bd	工作流执行 ID。
#{.custom-md-table-4}#

请求示例

Vid 模式触发工作流
以下示例演示如何以 GET 方式通过 Vid 触发工作流 06853553c4d3402698a17ff5dff87fd7。

https://vod.volcengineapi.com?Action=StartWorkflow&Version=2020-08-01&Vid=v0d399g10001csunlgaljhtddi6hgvn0&TemplateId=06853553c4d3402698a17ff5dff87fd7

DirectUrl 模式触发工作流
以下示例演示如何以 POST 方式通过文件路径 FileName 触发工作流 c3841b3122fd460db2bc99a6ec131cb8。

https://vod.volcengineapi.com?Action=StartWorkflow&Version=2020-08-01
{
  "DirectUrl": "{\"FileName\":\"vodtest.mp4\",\"SpaceName\":\"test-jinling\"}",
  "TemplateId": "c3841b3122fd460db2bc99a6ec131cb8"
}

动态设置截图文件路径
本示例演示如何在触发工作流 a378f66a5bc2495eb55fc14ac109625f（该工作流包含截图任务，关联了截图模板 94c8d47c4c39440b8e6b4e0ce7a96776）时，通过截图覆盖参数 Snapshot 动态设置截图文件路径为 {{fileTitle}}_{{templateId}}。假设片源文件名称为 vodtest，则此次截图生成的文件路径为 vodtest_a378f66a5bc2495eb55fc14ac109625f。

https://vod.volcengineapi.com?Action=StartWorkflow&Version=2020-08-01
{
  "Vid": "v0d399g10001csunlgaljhtddi6hgvn0",
  "TemplateId": "a378f66a5bc2495eb55fc14ac109625f",
  "Input": "{\"OverrideParams\": {\"Snapshot\": [{\"TemplateId\": [\"94c8d47c4c39440b8e6b4e0ce7a96776\"],\"FileName\": \"{{fileTitle}}_{{templateId}}\"}]}}"
}

返回示例
{
  "ResponseMetadata": {
    "RequestId": "202208121******514515206772F7C",
    "Action": "StartWorkflow",
    "Version": "2020-08-01",
    "Service": "vod",
    "Region": "cn-north-1"
  },
  "Result": {
    "RunId": "lb:0f557116******059cedf92f1bd"
  }
}
错误码
下表列举了本接口特有的错误码。如需了解更多错误码，详见视频点播公共错误码。

状态码	错误码	错误信息	说明
400	InvalidParameter	-	非法参数。
403	RequestForbidden	-	请求被禁止。
404	ResourceNotFound	-	ID 找不到。
409	ResourceInUse	-	ID 正在被使用（删除时）。
429	RequestLimitExceeded	-	请求超过上限。
500	InternalError	-	内部错误。
503	ServiceUnavailable	-	服务不可用。
服务端 SDK
视频点播为 OpenAPI 提供了配套的服务端 SDK，支持多种编程语言，帮助您实现快速开发。建议使用服务端 SDK 来调用 API。此 API 各语言调用的示例代码，请参考如下：

Java
Python
PHP
Go
Node.js
最近更新时间：2025.12.12 14:59:08
这个页面对您有帮助吗？
有用
无用
上一篇：
MGetMaterial - 批量获取素材
RetrieveTranscodeResult - 获取转码结果
下一篇
使用说明
注意事项
请求说明
调试
请求参数
DirectUrl
WorkflowParams
OverrideParams
LogoOverride
SnapshotOverride
TranscodeAudioOverride
TranscodeVideoOverride
Clip
返回参数
请求示例
Vid 模式触发工作流
DirectUrl 模式触发工作流
动态设置截图文件路径
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
