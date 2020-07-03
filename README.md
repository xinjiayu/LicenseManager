# LicenseManager
为golang的应用添加简单的license限制

这只是一个简单的日期限制，可以基于这个基础增加对目标服务器的IP、MAC地址等唯一标识进行限制


### 生成license授权码：
进入Createlic目录中，执行

`go build`

`./CreateLic -name XXXX应用名 -date 20221115`

`参数说明：`

name:应用名称
date:应用截至日期，格式为年月日，例如：20221230（这个时间是截至到2022年12月30日）

### 在应用在调用的方式：

```go
	LicenseManager.ValidAppLic("XXXX应用名", "app.lic")
```

`说明：`
XXXX应用名，授权验证的应用名称需要与生成授权码的应用名称是一致的。


#各种系统获取系统唯一ID的方式

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

