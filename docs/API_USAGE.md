# LicenseManager API 使用文档

## v1.2.0 新增：Ed25519 签名 API（推荐）

v2 许可证使用 Ed25519 非对称签名：签发方持有私钥，客户端只需内嵌公钥即可验证，无法伪造许可证。

### API 参考

| 函数 | 用途 |
|------|------|
| `GenerateSigningKeyPair() (privatePEM, publicPEM string, err error)` | 生成 Ed25519 密钥对（PKCS#8/PKIX PEM） |
| `EncryptLicSigned(appInfoFile, output, privateKeyPEM string) error` | 读取配置 JSON，签名并生成 v2 许可证 |
| `ValidAppLicSigned(licFile, publicKeyPEM string) (bool, error)` | 验证 v2 许可证（验签 + 设备 + 到期） |
| `GetLicenseInfoSigned(licFile, publicKeyPEM string) (*LicenseDisplayInfo, error)` | 获取 v2 许可证详细信息 |
| `GetLicenseInfoSignedFormatted(licFile, publicKeyPEM string) (string, error)` | 获取 v2 许可证的格式化文本 |

错误判定支持 `errors.Is`：`ErrInvalidSignature`（篡改/公钥不匹配）、`ErrLicenseExpired`（已过期）、`ErrDeviceMismatch`（设备不匹配）、`ErrClockRollback`（系统时间被回拨）。

### 使用示例（客户端验证）

```go
package main

import (
    "errors"
    "fmt"
    "log"

    "github.com/xinjiayu/LicenseManager"
)

// 公钥内嵌到客户端（从签发方获取）
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEA...
-----END PUBLIC KEY-----`

func main() {
    ok, err := LicenseManager.ValidAppLicSigned("app.lic", publicKeyPEM)
    switch {
    case err == nil && ok:
        fmt.Println("许可证有效")
    case errors.Is(err, LicenseManager.ErrInvalidSignature):
        fmt.Println("许可证被篡改或公钥不匹配:", err)
    case errors.Is(err, LicenseManager.ErrLicenseExpired):
        fmt.Println("许可证已过期:", err)
    case errors.Is(err, LicenseManager.ErrDeviceMismatch):
        fmt.Println("许可证不适用于此设备:", err)
    case errors.Is(err, LicenseManager.ErrClockRollback):
        fmt.Println("系统时间被回拨:", err)
    default:
        log.Fatal(err)
    }
}
```

## v1.1.0：许可证信息获取 API

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
    key := "your-16-byte-key"
    
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
    info, err := LicenseManager.GetLicenseInfo("app.lic", "your-16-byte-key")
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

```bash
# 文本格式显示
licensemanager show -f app.lic -k "your-key"

# JSON格式显示
licensemanager show -f app.lic -k "your-key" -format json

# v2签名格式许可证
licensemanager show -f app.lic -pubkey license_public_key.pem -format json
```

### 完整示例

参考 `examples/license_info/main.go` 文件查看完整的使用示例。 