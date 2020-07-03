# LicenseManager 授权管理

为golang的应用添加简单的license限制

这只是一个简单的日期限制，可以基于这个基础增加对目标服务器的IP、MAC地址等唯一标识进行限制


### license授权码生成工具：
进入Createlic目录中，执行

`go build` 编译License生成工具


`./CreateLic -h` 查看工具帮助信息

显示如下：

```
Usage: CreateLic [-hfk] [-f filename] [-k key]

Options:
  -f string
        需要授权的应用信息配置文件，json格式
  -h    查看帮助信息
  -k string
        授权钥匙 (default "1234567890123456")

```

license配置文件说明：
	
	AppName        string //应用名称
	AppCompany     string //应用发布的公司
	AppUUID        string //此次发布应用的UUID
	ObjUUID        string //目标设备的UUID
	AuthorizedName string //授权名称
	LimitedTime    string //到期日期
  
例子：

```bash
{
  "AppName":"LscServer",
  "AppCompany":"baidu company",
  "EncryptSalt":"2342423",
  "AppUUID":"556677888",
  "ObjUUID":"DB7074EB-7B7C-592F-823A-558FEC81051A",
  "AuthorizedName":"xxxx公司",
  "LimitedTime":"20221212"
}

```

### 在应用中调用的方式：

```go
	LicenseManager.ValidAppLic("AppName名", "key值")
```

**说明：**

AppName，授权验证的应用名称需要与生成授权码的应用名称是一致的。

key值，必须是16位的。并且与程序中使用的key值相同


### 各种操作系统，获取系统唯一ID的方式

BSD:

```source-shell
cat /etc/hostid
# or (might be empty)
kenv -q smbios.system.uuid
```

Linux:

```source-shell
cat /var/lib/dbus/machine-id
# or when not found (e.g. Fedora 20)
cat /etc/machine-id
```

OS X:

```source-shell
ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID
```

Windows:

```source-batchfile
reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography /v MachineGuid
```

or

* Open Windows Registry via `regedit`
* Navigate to `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography`
* Take value of key `MachineGuid`


