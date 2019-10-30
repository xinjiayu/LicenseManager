# LicenseManager
为golang的应用添加简单的license限制




###生成license授权码：
进入Createlic目录中，执行

`go build`

`./CreateLic -name XXXX应用名 -date 20221115`

`参数说明：`

name:应用名称
date:应用截至日期，格式为年月日，例如：20221230（这个时间是截至到2022年12月30日）

在应用在调用的方式：

```go
	LicenseManager.ValidAppLic("XXXX应用名", "app.lic")
```

`说明：`
XXXX应用名，授权验证的应用名称需要与生成授权码的应用名称是一致的。


