# certbot_tencent

当我们使用 certbot 申请通配符证书时，需要手动添加 TXT 记录。每个 certbot 申请的证书有效期为 3 个月，虽然 certbot 提供了自动续期命令，但是当我们把自动续期命令配置为定时任务时，我们无法手动添加新的 TXT 记录用于 certbot 验证。
好在 certbot 提供了一个 hook，可以编写一个脚本。在续期的时候让脚本调用 DNS 服务商的 API 接口动态添加 TXT 记录，验证完成后再删除此记录。
本项目使用go语言，编写了基于腾讯云动态添加和删除dns记录，达到了自动化更新证书的目的。 如果使用的aliyun,  可以搜索certbot-dns-aliyun 插件

前提：
1、使用certbot

2、已在https://console.dnspod.cn/domain 中获得SecretId和SecretKey

3、生成二进制可执行文件/etc/letsencrypt/tencent_dns

4、生成证书

`
certbot certonly -d yourdomain.com -d *.yourdomain.com --email youremail@qq.com --manual --manual-auth-hook "/etc/letsencrypt/tencent_dns auth"  --manual-cleanup-hook "/etc/letsencrypt/tencent_dns cleanup"  --preferred-challenges dns
`

5、续约
```certbot renew \
  --manual \
  --manual-auth-hook "/etc/letsencrypt/tencent_dns auth" \
  --manual-cleanup-hook "/etc/letsencrypt/tencent_dns cleanup" \
  --preferred-challenges dns \
  --deploy-hook "nginx -s reload"
```


  6、定时
  `crontab -e`
   `1 1 */1 * *  certbot renew --manual --manual-auth-hook "/etc/letsencrypt/tencent_dns auth"  --manual-cleanup-hook "/etc/letsencrypt/tencent_dns cleanup"  --preferred-challenges dns --deploy-hook "nginx -s reload"`

