# LicenseManager API 使用文档

## 新增功能：许可证信息获取

### 概述

LicenseManager v1.1.0 新增了两个重要的API函数，用于获取和显示许可证文件的详细信息：

1. `GetLicenseInfo(licenseFile, key string) (*LicenseDisplayInfo, error)` - 获取结构化的许可证信息
2. `GetLicenseInfoFormatted(licenseFile, key string) (string, error)` - 获取格式化的文本信息

### API 参考

#### 1. GetLicenseInfo

**函数签名**：
```go
func GetLicenseInfo(licenseFile, key string) (*LicenseDisplayInfo, error)
```

**参数**：
- `licenseFile`: 许可证文件路径
- `key`: 解密密钥

**返回值**：
- `*LicenseDisplayInfo`: 包含许可证详细信息的结构体
- `error`: 错误信息（通常为nil，错误信息在结构体中）

### 使用示例

#### 基本用法

```go
package main

import (
    "fmt"
    "log"
    "github.com/xinjiayu/LicenseManager"
)

func main() {
    licenseFile := "app.lic"
    key := "1234567890123456"
    
    // 获取结构化信息
    info, err := LicenseManager.GetLicenseInfo(licenseFile, key)
    if err != nil {
        log.Fatal(err)
    }
    
    if info.IsValid {
        fmt.Printf("✅ 许可证有效！应用: %s\n", info.AppName)
        fmt.Printf("剩余天数: %d\n", info.DaysRemaining)
    } else {
        fmt.Printf("❌ 许可证无效: %s\n", info.ErrorMessage)
    }
}
```

#### Web API 集成

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/xinjiayu/LicenseManager"
)

func licenseInfoHandler(w http.ResponseWriter, r *http.Request) {
    info, err := LicenseManager.GetLicenseInfo("app.lic", "your-secret-key")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(info)
}

func main() {
    http.HandleFunc("/license-info", licenseInfoHandler)
    http.ListenAndServe(":8080", nil)
}
```

### 命令行工具

#### ShowLicenseInfo 工具

```bash
# 文本格式显示
go run cmd/ShowLicenseInfo/main.go -f app.lic -k "your-key"

# JSON格式显示
go run cmd/ShowLicenseInfo/main.go -f app.lic -k "your-key" -format json
```

### 完整示例

参考 `examples/license_info/main.go` 文件查看完整的使用示例。 