package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

const (
	SecretID   = "AKIDBE5kC3333333333333333333" // 替换为你的SecretId
	SecretKey  = "RjohCXiiV7222222222222222222" // 替换为你的SecretKey
	Domain     = "habcde.com"                   // 主域名
	TTL        = 600                            // DNS记录TTL
	RecordLine = "默认"                           // 记录线路
)

var (
	client *dnspod.Client
)

func init() {
	credential := common.NewCredential(SecretID, SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	var err error
	client, err = dnspod.NewClient(credential, "", cpf)
	if err != nil {
		log.Fatalf("初始化客户端失败: %v", err)
	}
}

// 添加TXT记录
func addTxtRecord(subDomain, value string) (uint64, error) {
	request := dnspod.NewCreateRecordRequest()
	request.Domain = common.StringPtr(Domain)
	request.SubDomain = common.StringPtr(subDomain)
	request.RecordType = common.StringPtr("TXT")
	request.RecordLine = common.StringPtr(RecordLine)
	request.Value = common.StringPtr(value)
	request.TTL = common.Uint64Ptr(TTL)

	response, err := client.CreateRecord(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return 0, fmt.Errorf("API调用错误: %v", err)
	}
	if err != nil {
		return 0, fmt.Errorf("网络错误: %v", err)
	}

	if response.Response.RecordId == nil {
		return 0, fmt.Errorf("无法获取RecordId")
	}

	return *response.Response.RecordId, nil
}

// 删除TXT记录
func deleteTxtRecord(recordId uint64) error {
	request := dnspod.NewDeleteRecordRequest()
	request.Domain = common.StringPtr(Domain)
	request.RecordId = common.Uint64Ptr(recordId)

	_, err := client.DeleteRecord(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return fmt.Errorf("API调用错误: %v", err)
	}
	if err != nil {
		return fmt.Errorf("网络错误: %v", err)
	}

	return nil
}

// 查询TXT记录ID
func findTxtRecord(subDomain string) (uint64, error) {
	request := dnspod.NewDescribeRecordListRequest()
	request.Domain = common.StringPtr(Domain)
	request.RecordType = common.StringPtr("TXT")

	response, err := client.DescribeRecordList(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return 0, fmt.Errorf("API调用错误: %v", err)
	}
	if err != nil {
		return 0, fmt.Errorf("网络错误: %v", err)
	}

	for _, record := range response.Response.RecordList {
		if *record.Name == subDomain {
			return *record.RecordId, nil
		}
	}

	return 0, fmt.Errorf("未找到记录")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("用法: ./tencent-dns-auth [auth|cleanup]")
	}

	switch os.Args[1] {
	case "auth":
		// 从环境变量获取 Certbot 传递的参数
		domain := os.Getenv("CERTBOT_DOMAIN")
		validation := os.Getenv("CERTBOT_VALIDATION")

		if domain == "" || validation == "" {
			log.Fatal("缺少必要的环境变量: CERTBOT_DOMAIN 或 CERTBOT_VALIDATION")
		}

		// 根据域名确定子域名
		subDomain := "_acme-challenge"
		if domain != Domain {
			// 处理子域名情况，如 test.hqfupai.com
			subDomain = "_acme-challenge." + domain[:len(domain)-len(Domain)-1]
		}

		recordId, err := addTxtRecord(subDomain, validation)
		if err != nil {
			log.Fatalf("添加TXT记录失败: %v", err)
		}
		log.Printf("成功添加TXT记录，RecordId: %d", recordId)

		// 保存RecordId到临时文件，供cleanup使用
		if err := os.WriteFile(fmt.Sprintf("/tmp/certbot_%s", domain), []byte(fmt.Sprintf("%d", recordId)), 0644); err != nil {
			log.Printf("警告: 无法保存RecordId到临时文件: %v", err)
		}

		// 等待DNS生效
		log.Println("等待30秒让DNS记录生效...")
		time.Sleep(30 * time.Second)

	case "cleanup":
		domain := os.Getenv("CERTBOT_DOMAIN")
		if domain == "" {
			log.Fatal("缺少环境变量: CERTBOT_DOMAIN")
		}

		// 尝试从临时文件读取RecordId
		data, err := os.ReadFile(fmt.Sprintf("/tmp/certbot_%s", domain))
		var recordId uint64
		if err == nil {
			_, err = fmt.Sscanf(string(data), "%d", &recordId)
			if err != nil {
				log.Printf("无法从临时文件解析RecordId: %v", err)
			}
		}

		if err != nil {
			// 回退到查询记录
			subDomain := "_acme-challenge"
			if domain != Domain {
				subDomain = "_acme-challenge." + domain[:len(domain)-len(Domain)-1]
			}
			recordId, err = findTxtRecord(subDomain)
			if err != nil {
				log.Printf("未找到要删除的记录: %v", err)
				return
			}
		}

		if err := deleteTxtRecord(recordId); err != nil {
			log.Fatalf("删除TXT记录失败: %v", err)
		}
		log.Printf("成功删除TXT记录，RecordId: %d", recordId)

		// 清理临时文件
		os.Remove(fmt.Sprintf("/tmp/certbot_%s", domain))

	default:
		log.Fatal("无效参数，用法: ./tencent-dns-auth [auth|cleanup]")
	}
}
