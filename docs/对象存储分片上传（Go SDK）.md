
文档中心
火山方舟大模型服务平台
豆包语音
扣子
云服务器
API网关
火山方舟大模型服务平台
豆包语音
扣子
云服务器
API网关
文档
备案
控制台
登录
立即注册
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

上传对象

分片上传（Go SDK）

复制全文
我的收藏
分片上传（Go SDK）
上传大对象时可以分成多个数据块（part）来分别上传，最后调用合并分片将上传的数据块合并为一个对象。​
注意事项​
分片上传前，您必须具有 tos:PutObject 权限，具体操作，请参见权限配置指南。​
取消分片上传任务前，您必须具有 tos:AbortMultipartUpload 权限，具体操作，请参见权限配置指南。​
分片编号从 1 开始，最大为 10000。除最后一个分片以外，其他分片大小最小为 4MiB。​
上传对象时，对象名必须满足一定规范，详细信息，请参见对象命名规范。​
TOS 是面向海量存储设计的分布式对象存储产品，内部分区存储了对象索引数据，为横向扩展您上传对象和下载对象时的最大吞吐量，和减小热点分区的概率，请您避免使用字典序递增的对象命名方式，详细信息，请参见性能优化。​
如果桶中已经存在同名对象，则新对象会覆盖已有的对象。如果您的桶开启了版本控制，则会保留原有对象，并生成一个新版本号用于标识新上传的对象。​
分片上传步骤​
分片上传一般包含以下三个步骤：​
初始化分片上传任务：调用 CreateMultipartUploadV2 方法返回 TOS 创建的全局唯一 UploadID。​
上传分片：调用 UploadPartV2 方法上传分片数据。​
说明​
对于同一个分片上传任务(通过 UploadID 标识)，分片编号(PartNumber)标识了该分片在整个对象中的相对位置。若通过同一分片编号多次上传数据，TOS 中会覆盖原始数据，并以最后一次上传数据为准。​
响应头中包含了数据的 MD5 值，可通过 Etag 获取。合并分片时，您需指定当前分片上传任务中的所有分片信息(分片编号、ETag 值)。​
完成分片上传：所有分片上传完成后，调用 CompleteMultipartUploadV2 方法将所有分片合并成一个完整的对象。​
示例代码​
分片上传完整过程​
下面代码展示将本地文件通过分片的方式上传完整过程，并在上传时指定 ACL 为 Private，存储类型为低频存储以及添加自定义元数据。​
​
package main​
​
import (​
   "context"​
   "fmt"​
   "io"​
   "os"​
​
   "github.com/volcengine/ve-tos-golang-sdk/v2/tos"​
   "github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"​
)​
​
func checkErr(err error) {​
   if err != nil {​
      if serverErr, ok := err.(*tos.TosServerError); ok {​
         fmt.Println("Error:", serverErr.Error())​
         fmt.Println("Request ID:", serverErr.RequestID)​
         fmt.Println("Response Status Code:", serverErr.StatusCode)​
         fmt.Println("Response Header:", serverErr.Header)​
         fmt.Println("Response Err Code:", serverErr.Code)​
         fmt.Println("Response Err Msg:", serverErr.Message)​
      } else if clientErr, ok := err.(*tos.TosClientError); ok {​
         fmt.Println("Error:", clientErr.Error())​
         fmt.Println("Client Cause Err:", clientErr.Cause.Error())​
      } else {​
         fmt.Println("Error:", err)​
      }​
      panic(err)​
   }​
}​
​
func main() {​
   var (​
      accessKey = os.Getenv("TOS_ACCESS_KEY")​
      secretKey = os.Getenv("TOS_SECRET_KEY")​
      // Bucket 对应的 Endpoint，以华北2（北京）为例：https://tos-cn-beijing.volces.com​
      endpoint = "https://tos-cn-beijing.volces.com"​
      region   = "cn-beijing"​
      // 填写 BucketName​
      bucketName = "*** Provide your bucket name ***"​
​
      // 指定的 ObjectKey​
      objectKey = "*** Provide your object name ***"​
      ctx       = context.Background()​
   )​
   // 初始化客户端​
   client, err := tos.NewClientV2(endpoint, tos.WithRegion(region), tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)))​
   checkErr(err)​
   // 初始化分片，指定对象权限为私有，存储类型为标准存储并设置元数据信息​
​
   createMultipartOutput, err := client.CreateMultipartUploadV2(ctx, &tos.CreateMultipartUploadV2Input{​
      Bucket:       bucketName,​
      Key:          objectKey,​
      ACL:          enum.ACLPrivate,​
      StorageClass: enum.StorageClassStandard,​
      Meta:         map[string]string{"key": "value"},​
   })​
   checkErr(err)​
   fmt.Println("CreateMultipartUploadV2 Request ID:", createMultipartOutput.RequestID)​
   // 获取到上传的 UploadID​
   fmt.Println("CreateMultipartUploadV2 Upload ID:", createMultipartOutput.UploadID)​
   // 需要上传的文件路径​
   localFile := "/root/example.txt"​
   fd, err := os.Open(localFile)​
   checkErr(err)​
   defer fd.Close()​
   stat, err := os.Stat(localFile)​
   checkErr(err)​
   fileSize := stat.Size()​
   // partNumber 编号从 1 开始​
   partNumber := 1​
   // part size 大小设置为 20M​
   partSize := int64(20 * 1024 * 1024)​
   offset := int64(0)​
   var parts []tos.UploadedPartV2​
   for offset < fileSize {​
      uploadSize := partSize​
      // 最后一个分片​
      if fileSize-offset < partSize {​
         uploadSize = fileSize - offset​
      }​
      fd.Seek(offset, io.SeekStart)​
      partOutput, err := client.UploadPartV2(ctx, &tos.UploadPartV2Input{​
         UploadPartBasicInput: tos.UploadPartBasicInput{​
            Bucket:     bucketName,​
            Key:        objectKey,​
            UploadID:   createMultipartOutput.UploadID,​
            PartNumber: partNumber,​
         },​
         Content:       io.LimitReader(fd, uploadSize),​
         ContentLength: uploadSize,​
      })​
      checkErr(err)​
      fmt.Println("upload Request ID:", partOutput.RequestID)​
      parts = append(parts, tos.UploadedPartV2{PartNumber: partNumber, ETag: partOutput.ETag})​
      offset += uploadSize​
      partNumber++​
   }​
​
   completeOutput, err := client.CompleteMultipartUploadV2(ctx, &tos.CompleteMultipartUploadV2Input{​
      Bucket:   bucketName,​
      Key:      objectKey,​
      UploadID: createMultipartOutput.UploadID,​
      Parts:    parts,​
   })​
   checkErr(err)​
   fmt.Println("CompleteMultipartUploadV2 Request ID:", completeOutput.RequestID)​
​
}​
​
列举已上传分片​
以下代码用于列举指定存储桶中指定对象已上传的分片信息。​
​
package main​
​
import (​
   "context"​
   "fmt"​
​
   "github.com/volcengine/ve-tos-golang-sdk/v2/tos"​
)​
​
func checkErr(err error) {​
   if err != nil {​
      if serverErr, ok := err.(*tos.TosServerError); ok {​
         fmt.Println("Error:", serverErr.Error())​
         fmt.Println("Request ID:", serverErr.RequestID)​
         fmt.Println("Response Status Code:", serverErr.StatusCode)​
         fmt.Println("Response Header:", serverErr.Header)​
         fmt.Println("Response Err Code:", serverErr.Code)​
         fmt.Println("Response Err Msg:", serverErr.Message)​
      } else if clientErr, ok := err.(*tos.TosClientError); ok {​
         fmt.Println("Error:", clientErr.Error())​
         fmt.Println("Client Cause Err:", clientErr.Cause.Error())​
      } else {​
         fmt.Println("Error:", err)​
      }​
      panic(err)​
   }​
}​
​
func main() {​
   var (​
      accessKey = os.Getenv("TOS_ACCESS_KEY")​
      secretKey = os.Getenv("TOS_SECRET_KEY")​
      // Bucket 对应的 Endpoint，以华北2（北京）为例：https://tos-cn-beijing.volces.com​
      endpoint = "https://tos-cn-beijing.volces.com"​
      region   = "cn-beijing"​
      // 填写 BucketName​
      bucketName = "*** Provide your bucket name ***"​
​
      // 指定的 ObjectKey​
      objectKey = "*** Provide your object name ***"​
      uploadID  = "*** Provide upload ID ***"​
      ctx       = context.Background()​
   )​
   // 初始化客户端​
   client, err := tos.NewClientV2(endpoint, tos.WithRegion(region), tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)))​
   checkErr(err)​
   // 列举 uploadID 已上传分片信息​
   truncated := true​
   marker := 0​
   for truncated {​
      output, err := client.ListParts(ctx, &tos.ListPartsInput{​
         Bucket:           bucketName,​
         Key:              objectKey,​
         UploadID:         uploadID,​
         PartNumberMarker: marker,​
      })​
      checkErr(err)​
      truncated = output.IsTruncated​
      marker = output.NextPartNumberMarker​
      for _, part := range output.Parts {​
         fmt.Println("Part Number:", part.PartNumber)​
         fmt.Println("ETag:", part.ETag)​
         fmt.Println("Size:", part.Size)​
      }​
   }​
​
}​
​
取消分片上传任务​
您可以通过 AbortMultipartUpload 方法来取消分片上传任务。当一个分片任务被取消后， TOS 会将已上传的分片数据删除，同时您无法再对此分片任务进行任何操作。​
​
package main​
​
import (​
   "context"​
   "fmt"​
​
   "github.com/volcengine/ve-tos-golang-sdk/v2/tos"​
)​
​
func checkErr(err error) {​
   if err != nil {​
      if serverErr, ok := err.(*tos.TosServerError); ok {​
         fmt.Println("Error:", serverErr.Error())​
         fmt.Println("Request ID:", serverErr.RequestID)​
         fmt.Println("Response Status Code:", serverErr.StatusCode)​
         fmt.Println("Response Header:", serverErr.Header)​
         fmt.Println("Response Err Code:", serverErr.Code)​
         fmt.Println("Response Err Msg:", serverErr.Message)​
      } else if clientErr, ok := err.(*tos.TosClientError); ok {​
         fmt.Println("Error:", clientErr.Error())​
         fmt.Println("Client Cause Err:", clientErr.Cause.Error())​
      } else {​
         fmt.Println("Error:", err)​
      }​
      panic(err)​
   }​
}​
​
func main() {​
   var (​
      accessKey = os.Getenv("TOS_ACCESS_KEY")​
      secretKey = os.Getenv("TOS_SECRET_KEY")​
      // Bucket 对应的 Endpoint，以华北2（北京）为例：https://tos-cn-beijing.volces.com​
      endpoint = "https://tos-cn-beijing.volces.com"​
      region   = "cn-beijing"​
      // 填写 BucketName​
      bucketName = "*** Provide your bucket name ***"​
​
      // 指定的 ObjectKey​
      objectKey = "*** Provide your object name ***"​
      uploadID  = "*** Provide upload ID ***"​
      ctx       = context.Background()​
   )​
   // 初始化客户端​
   client, err := tos.NewClientV2(endpoint, tos.WithRegion(region), tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)))​
   checkErr(err)​
   // 取消分片上传​
   output, err := client.AbortMultipartUpload(ctx, &tos.AbortMultipartUploadInput{​
      Bucket:   bucketName,​
      Key:      objectKey,​
      UploadID: uploadID,​
   })​
   checkErr(err)​
   fmt.Println("AbortMultipartUpload Request ID:", output.RequestID)​
}​
​
相关文档​
关于创建分片上传任务的 API 文档，请参见 CreateMultipartUpload。​
关于上传分片的 API 文档，请参见 UploadPart。​
关于合并分片的 API 文档，请参见 CompleteMultipartUpload。​
关于取消分片上传任务的 API 文档，请参见 AbortMultipartUpload。​
​
最近更新时间：2025.10.17 16:03:17
这个页面对您有帮助吗？
有用
无用
上一篇：
追加上传（Go SDK）
断点续传（Go SDK）
下一篇
注意事项
分片上传步骤
示例代码
分片上传完整过程
列举已上传分片
取消分片上传任务
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

