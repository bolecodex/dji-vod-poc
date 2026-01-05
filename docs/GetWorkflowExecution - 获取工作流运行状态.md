
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

媒体处理

GetWorkflowExecution - 获取工作流运行状态

复制全文
我的收藏
GetWorkflowExecution - 获取工作流运行状态
调用 GetWorkflowExecution 接口指定 RunId 工作流任务 ID，获取工作流运行状态。

注意事项
目前支持查询任务的时间范围为 30 天。
本接口的单用户 QPS 限制为 50 次/秒。超过限制，API 调用会被限流，这可能会影响您的业务，请合理调用。更多信息，请参见 QPS 限制。
请求说明
请求地址：https://vod.volcengineapi.com?Action=GetWorkflowExecution&Version=2020-08-01

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
GetWorkflowExecution
接口名称。当前 API 的名称为 GetWorkflowExecution。
Version
String
是
2020-08-01
接口版本。当前 API 的版本为 2020-08-01。
RunId
String
是
fershg899***
工作流任务 ID。用于唯一指示当前这次转码事件。可通过触发工作流接口获取。
返回参数
下表仅列出本接口特有的返回参数。更多信息请见公共返回参数。

参数
类型
示例值
描述
RunId
String
2ce437f301a9***cf80b5369fe4
工作流任务 ID。
Vid
String
v0c931g7007ac***833096m05os10
视频 ID。
DirectUrl
Object
-
DirectUrl 模式下文件存储路径。
SpaceName
String
test
空间名称。可调用 ListSpace 获取当前账号下所有空间的信息。
FileName
String
example.mp4
文件路径。音视频上传至视频点播服务后，您可通过媒资上传完成事件获取 FileName。
BucketName
String
tos-cn-v-cd3a11
存储桶名称。可调用 ListSpace 获取空间所绑定的存储桶名称。
TemplateId
String
3447e8db6***6271401b132a455
工作流模板 ID。
SpaceName
String
test
点播空间名称。
TemplateName
String
模板组-hls
工作流模板名称。
Status
String
0
执行状态。取值如下：

工作流任务执行完成后，取值如下：

0：成功。
[1000,1999]：用户错误的失败。
[2000,2999]：系统错误的失败。
说明

具体详见工作流状态码。

工作流任务执行过程中，取值如下：

PendingStart：排队中。
Running：执行中。
用户终止操作，工作流任务执行结束，取值如下：

Terminated：终止。
EnableLowPriority
Boolean
true
是否为闲时任务。取值如下：

false：正常任务。
true：闲时任务。
JobSource
String
API
任务来源。取值如下：

API：主动触发。
AutoTrigger：自动触发。
TranscodeStrategy：策略触发。
SystemTrigger：系统触发。
CreateTime
String
2022-08-29T03:30:04Z
创建时间。遵循 RFC3339 格式的 UTC 时间，精度为秒，格式为：yyyy-MM-ddTHH:mm:ssZ。
StartTime
String
2022-08-29T04:46:22Z
开始时间。遵循 RFC3339 格式的 UTC 时间，精度为秒，格式为：yyyy-MM-ddTHH:mm:ssZ。
EndTime
String
2022-08-29T04:46:22Z
结束时间。遵循 RFC3339 格式的 UTC 时间，精度为秒，格式为：yyyy-MM-ddTHH:mm:ssZ。
请求示例
https://vod.volcengineapi.com?Action=GetWorkflowExecution&Version=2020-08-01&RunId=fershg899***
返回示例
{
  "ResponseMetadata": {
    "RequestId": "2022090515143***2551810746A0D7",
    "Action": "GetWorkflowExecution",
    "Version": "2020-08-01",
    "Service": "vod",
    "Region": "cn-north-1"
  },
  "Result": {
    "RunId": "2ce437f301a9***cf80b5369fe4",
    "Vid": "v0c931g7007ac***833096m05os10",
    "TemplateId": "3447e8db6***6271401b132a455",
    "TemplateName": "模板组-hls",
    "SpaceName": "000000",
    "Status": "0",
    "TaskListId": "87f133ea9b***d3058335f673",
    "JobSource": "API",
    "CreateTime": "2022-08-29T03:30:04Z",
    "StartTime": "2022-08-29T04:46:22Z",
    "EndTime": "2022-08-29T04:46:22Z",
    "EnableLowPriority": true
  }
}
错误码
下表列举了本接口特有的错误码。如需了解更多错误码，详见视频点播公共错误码。

状态码	错误码	错误信息	说明
400	InvalidParameter	-	非法参数。
403	RequestForbidden	-	非法请求。
404	ResourceNotFound	-	ID 找不到。
409	ResourceInUse	-	ID 正在被使用（删除时）。
429	RequestLimitExceeded	-	请求超过上限。
500	InternalError	-	内部错误。
503	ServiceUnavailable	-	服务不可用。
服务端 SDK
点播 OpenAPI 提供了配套的服务端 SDK，支持多种编程语言，帮助您实现快速开发。建议使用服务端 SDK 来调用 API，此 API 各语言调用的示例代码，请参考如下：

Java
Python
PHP
Go
最近更新时间：2025.02.13 15:45:03
这个页面对您有帮助吗？
有用
无用
上一篇：
RetrieveTranscodeResult - 获取转码结果
GetWorkflowExecutionResult - 获取工作流执行结果
下一篇
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
