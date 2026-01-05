
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
对象存储
文档指南
工具指南
API 参考
SDK 参考
请输入

文档首页

对象存储

Go

对象接口

管理对象

重命名对象（Go SDK）

复制全文
我的收藏
重命名对象（Go SDK）
TOS 支持在桶内 Rename 单个对象的 Key，不拷贝和删除数据。

注意事项
重命名对象前，您需要先开启重命名功能。
仅支持重命名开启 RenameObject 后新上传的对象，不支持重命名开启该功能前的存量对象。
同一个对象不支持并发重命名。
重命名对象元数据上的所有信息都与源对象一致。更多信息，请参见使用 RenameObject。

示例代码
以下代码用于对象重命名功能。

package main

import (
    "context"
    "fmt"
    "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

func checkErr(err error) {
    if err != nil {
       if serverErr, ok := err.(*tos.TosServerError); ok {
          fmt.Println("Error:", serverErr.Error())
          fmt.Println("Request ID:", serverErr.RequestID)
          fmt.Println("Response Status Code:", serverErr.StatusCode)
          fmt.Println("Response Header:", serverErr.Header)
          fmt.Println("Response Err Code:", serverErr.Code)
          fmt.Println("Response Err Msg:", serverErr.Message)
       } else {
          fmt.Println("Error:", err)
       }
       panic(err)
    }
}

func main() {
    var (
       accessKey = os.Getenv("TOS_ACCESS_KEY")
       secretKey = os.Getenv("TOS_SECRET_KEY")
       // Bucket 对于的 Endpoint，以华北2（北京）为例：https://tos-cn-beijing.volces.com
       endpoint = "https://tos-cn-beijing.volces.com"
       region   = "cn-beijing"
       // 填写 BucketName
       bucketName = "*** Provide your bucket name ***"
       key        = "old-key"
       newKey     = "new-key"
       ctx        = context.Background()
    )
    // 初始化客户端
    client, err := tos.NewClientV2(endpoint, tos.WithRegion(region), tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)))
    checkErr(err)
    // 重命名对象
    output, err := client.RenameObject(ctx, &tos.RenameObjectInput{
       Bucket: bucketName,
       Key:    key,
       NewKey: newKey,
    })
    checkErr(err)
    fmt.Println("RenameObject Request ID:", output.RequestID)
}

相关文档
关于 RenameObject 的 API 文档，请参见 RenameObject。

最近更新时间：2024.06.20 20:04:18
这个页面对您有帮助吗？
有用
无用
上一篇：
禁止覆盖同名文件（Go SDK）
设置对象级过期时间（Go SDK）
下一篇
注意事项
示例代码
相关文档

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
