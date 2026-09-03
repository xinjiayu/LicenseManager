# LicenseManager v1.2.0 - 统一许可证管理工具

## 🎯 概述

LicenseManager 是一个功能完整的许可证管理系统，用于生成、验证和管理软件许可证。v1.2.0 版本新增 **Ed25519 非对称签名**（v2 许可证格式），客户端只需内嵌公钥即可验证许可证，从根本上解决了对称加密下"验证密钥即可伪造许可证"的问题。

## ✨ 主要特性

- 🔏 **Ed25519 签名（v2，推荐）**：私钥签发、公钥验证，客户端无法伪造许可证
- 🔐 **AES加密（v1，兼容）**：支持16/24/32字节密钥，使用随机IV
- 📱 **设备绑定**：基于设备UUID的许可证绑定机制
- ⏰ **到期管理**：支持许可证过期时间设置和剩余天数计算
- 🛠️ **统一工具**：一个命令行工具包含所有功能（create/verify/show/checkuuid/genkeys）
- 📊 **多种输出**：支持文本和JSON格式输出
- 🔍 **详细信息**：完整的许可证信息查看功能

## 🚀 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/xinjiayu/LicenseManager.git
cd LicenseManager

# 编译统一工具
go build -o licensemanager ./cmd/licensemanager

# 或者直接运行
go run ./cmd/licensemanager help
```

### 基本使用（v2 Ed25519 签名，推荐）

```bash
# 1. 生成 Ed25519 密钥对（一次性操作，私钥妥善保管）
./licensemanager genkeys -o ./keys

# 2. 获取设备UUID（发给签发方）
./licensemanager checkuuid

# 3. 签发许可证（签发方持有私钥）
./licensemanager create -f config.json -privkey keys/license_private_key.pem -o app.lic

# 4. 验证许可证（客户端只需公钥）
./licensemanager verify -f app.lic -pubkey keys/license_public_key.pem

# 5. 查看许可证信息
./licensemanager show -f app.lic -pubkey keys/license_public_key.pem -format json
```

### 基本使用（v1 AES 对称加密，兼容旧版）

```bash
./licensemanager create -f config.json -k "your-16-byte-key"
./licensemanager verify -f app.lic -k "your-16-byte-key"
./licensemanager show -f app.lic -k "your-16-byte-key"
```

## 🔏 Ed25519 非对称签名（v2 格式）

### 为什么需要非对称签名

v1 使用 AES 对称加密：客户端要验证许可证就必须内嵌解密密钥，而**持有解密密钥就意味着可以自己加密生成任意许可证**——逆向客户端二进制拿到密钥后即可无限伪造。v2 使用 Ed25519 签名：私钥只存在于签发方，客户端仅内嵌公钥；公钥只能验证、不能签发，任何对许可证内容的篡改（改到期时间、换设备UUID、换应用名）都会导致验签失败。

### v2 许可证文件格式

```json
{
  "version": 2,
  "alg": "Ed25519",
  "issued_at": "2026-09-04T00:23:02+08:00",
  "payload": "base64编码的授权信息JSON",
  "signature": "base64编码的Ed25519签名（覆盖payload解码后的原始字节）"
}
```

> 说明：v2 格式的授权内容是"签名保护的明文"而非加密密文。许可证需要的是防篡改而非保密——被授权设备上本就能读到这些信息。如确有保密需求，可在签名前自行对 payload 做一层加密。

### 密钥管理建议

- 私钥（`license_private_key.pem`）：只在签发许可证的机器上保存，权限 0600，可离线保管
- 公钥（`license_public_key.pem`）：内嵌到客户端程序（硬编码常量或随二进制分发）
- 密钥遗失/泄露时的处理：换新密钥对重新签发，客户端升级时更新内嵌公钥

### 通用许可证与自定义分级校验

- **通用许可证**（不绑定设备，适合内嵌试用授权、批量延期）：签发用 `EncryptLicSignedUniversal` 或 CLI `create -universal`（须与 `-privkey` 组合）。库层安全底线：禁止签发永久通用许可证，LimitedTime 必填；有效期上限等产品规则由接入方自行叠加。
- **仅验签解析**：`ParseSignedLicenseFile`（文件）/ `ParseSignedLicenseContent`（字节，适配 `-ldflags` 内嵌授权）只做 Ed25519 验签与内容解析，不做设备/到期/回拨判定，供接入方实现自定义分级（如通用授权跳过设备绑定）。
- **回拨状态路径**：`SetClockStatePath(path)` 在进程初始化期设置防回拨状态文件路径（容器部署建议指向持久卷；传空串显式禁用该机制）。

### Go API（v2 签名许可证）

```go
package main

import (
    "fmt"
    "log"

    "github.com/xinjiayu/LicenseManager"
)

func main() {
    // 签发端：生成密钥对（一次性）
    privatePEM, publicPEM, err := LicenseManager.GenerateSigningKeyPair()
    if err != nil {
        log.Fatal(err)
    }
    // privatePEM 妥善保存，publicPEM 内嵌到客户端 ...

    // 签发端：生成签名许可证
    if err := LicenseManager.EncryptLicSigned("config.json", "app.lic", privatePEM); err != nil {
        log.Fatal(err)
    }

    // 客户端：验证签名许可证（只需公钥）
    ok, err := LicenseManager.ValidAppLicSigned("app.lic", publicPEM)
    if err != nil || !ok {
        log.Fatalf("许可证无效: %v", err)
    }

    // 客户端：获取详细信息
    info, err := LicenseManager.GetLicenseInfoSigned("app.lic", publicPEM)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("应用: %s, 状态: %s, 剩余: %d天\n", info.AppName, info.Status, info.DaysRemaining)
}
```

## 📖 详细功能（v1/v2 通用说明）

### 1. 创建许可证 (create)

从JSON配置文件生成许可证文件，推荐使用 v2 签名格式。

```bash
# v2 Ed25519签名格式（推荐）
licensemanager create -f config.json -privkey license_private_key.pem

# v1 AES加密格式（已不推荐，-k 必须显式传入16/24/32字节密钥）
licensemanager create -f config.json -k "your-32-byte-secret-key" -o production.lic
```

**配置文件格式**：
```json
{
  "AppName": "YourApp",
  "AppCompany": "Your Company Ltd",
  "AppUUID": "app-unique-identifier",
  "ObjUUID": "device-uuid-from-checkuuid-command",
  "AuthorizedName": "客户公司名称",
  "LimitedTime": "20261231",
  "LicenseID": "LIC202501001",
  "LicenseQuantity": 100
}
```

### 2. 验证许可证 (verify)

验证许可证文件的有效性，包括：
- 签名验证（v2）或解密校验（v1）
- 设备UUID匹配验证
- 过期时间检查（按UTC）
- 系统时间回拨检测

```bash
# v2 签名格式（推荐）
licensemanager verify -f app.lic -pubkey license_public_key.pem

# v1 AES格式（已不推荐）
licensemanager verify -f app.lic -k "your-16-byte-key"
```

### 3. 显示许可证信息 (show/info)

获取许可证的详细信息，支持文本和JSON格式。

```bash
# v2 签名格式（推荐）
licensemanager show -f app.lic -pubkey license_public_key.pem

# JSON格式（适合API集成）
licensemanager show -f app.lic -pubkey license_public_key.pem -format json
```

**JSON输出示例**：
```json
{
  "app_name": "YourApp",
  "app_company": "Your Company Ltd",
  "authorized_name": "客户公司名称",
  "limited_time": "20261231",
  "license_id": "LIC202501001",
  "license_quantity": 100,
  "status": "valid",
  "days_remaining": 365,
  "is_valid": true,
  "error_message": ""
}
```

### 4. 获取设备UUID (checkuuid/uuid)

获取当前设备的唯一标识符，用于生成设备绑定的许可证。

```bash
licensemanager checkuuid
# 输出: F6235A40-C9E2-5681-B236-ED9C4C15E58D
```

### 5. 生成签名密钥对 (genkeys/genkey)

生成 Ed25519 密钥对，用于 v2 签名格式许可证的签发与验证。

```bash
licensemanager genkeys -o ./keys
# 输出:
#   keys/license_private_key.pem  私钥（签发用，权限0600，妥善保管）
#   keys/license_public_key.pem   公钥（内嵌到客户端验证）
```

## 🔧 API 使用

> 以下为 v1 AES 格式的 API。**新接入请使用上文 v2 签名 API**（`ValidAppLicSigned` 等），
> v1 API 仅为兼容存量许可证保留。

### Go API 集成

```go
package main

import (
    "fmt"
    "github.com/xinjiayu/LicenseManager"
)

func main() {
    // 获取许可证信息
    info, err := LicenseManager.GetLicenseInfo("app.lic", "your-16-byte-key")
    if err != nil {
        panic(err)
    }
    
    if info.IsValid {
        fmt.Printf("应用: %s, 剩余: %d天\n", info.AppName, info.DaysRemaining)
    } else {
        fmt.Printf("许可证无效: %s\n", info.ErrorMessage)
    }
    
    // 验证许可证
    isValid, err := LicenseManager.ValidAppLic("app.lic", "your-16-byte-key")
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }
    
    fmt.Printf("许可证状态: %v\n", isValid)
}
```

### Web API 示例

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/xinjiayu/LicenseManager"
)

func licenseHandler(w http.ResponseWriter, r *http.Request) {
    info, err := LicenseManager.GetLicenseInfo("app.lic", "secret-key")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(info)
}

func main() {
    http.HandleFunc("/license", licenseHandler)
    http.ListenAndServe(":8080", nil)
}
```

## 🔐 安全特性

1. **Ed25519签名（v2）**：私钥签发、公钥验证，客户端无法伪造或篡改许可证
2. **时间回拨防护**：本地记录每个许可证最后通过验证的时间（只增不减），系统时间被明显回拨（超过48小时）时拒绝验证；状态文件损坏不阻塞验证
3. **设备绑定**：许可证与特定设备UUID绑定
4. **到期管理**：到期比较与剩余天数统一按UTC计算，避免跨时区偏差
5. **完整性检查**：签名验证（v2）与PKCS7填充校验（v1）确保数据完整性
6. **AES加密（v1，已弃用）**：随机IV；因客户端内嵌的解密密钥可被用于伪造许可证，v1仅保留用于兼容存量许可证，新签发请使用v2

### 已知局限（离线许可证的固有边界）

- **防回拨是尽力而为**：状态文件（`~/.licensemanager/clock_state.json`）可被删除；从未在本机验证过的过期许可证回拨后仍可通过。高价值场景建议配合在线激活。
- **machineid可被伪造**：Windows注册表/部分平台文件型machine-id可被篡改或克隆。如需更强设备指纹，可自行组合硬件序列号、MAC等，或在 `AppLicenseInfo.ObjUUID` 中使用自采指纹。
- **公钥分发**：内嵌在客户端二进制中的公钥可能被连锅端替换（攻击者同时替换许可证与公钥常量并重打包）；混淆/加壳可提高门槛，根因解决需在线验证。

## 📁 项目结构

```
LicenseManager/
├── cmd/
│   └── licensemanager/          # 统一命令行工具
│       ├── main.go             # 主程序入口
│       └── README.md           # 工具使用说明
├── utils/
│   ├── aes.go                  # AES加密解密工具
│   └── aes_test.go             # AES单元测试
├── examples/
│   └── license_info/           # 使用示例
├── lic.go                      # 核心许可证功能（v1 AES格式）
├── sign.go                     # Ed25519签名功能（v2签名格式）
├── clock.go                    # 系统时间回拨防护
├── lic_test.go                 # v1单元测试
├── lic_sign_test.go            # v2签名单元测试
├── clock_test.go               # 回拨防护单元测试
├── go.mod                      # Go模块定义
├── LICENSE                     # MIT许可证
└── README.md                   # 本文档
```

## 🧪 测试

```bash
# 运行全部单元测试（自包含，自动适配本机UUID，无需预先准备文件）
go test ./... -v

# 测试完整工作流程
go build -o licensemanager ./cmd/licensemanager

# 生成签名密钥对并测试v2签名流程
./licensemanager genkeys -o ./keys
./licensemanager create -f config.json -privkey keys/license_private_key.pem -o app.lic
./licensemanager verify -f app.lic -pubkey keys/license_public_key.pem
./licensemanager show -f app.lic -pubkey keys/license_public_key.pem -format json

# 测试v1 AES流程（兼容旧版，需显式传密钥）
./licensemanager create -f config.json -k "your-16-byte-key" -o v1.lic
./licensemanager verify -f v1.lic -k "your-16-byte-key"
```

## 🔄 迁移指南

### 从v1.x升级到v1.2.0

1. **API兼容**：原有函数保持不变；`EncryptLic` 已标记 Deprecated，请改用 `EncryptLicToFile`
2. **文件兼容**：v1许可证文件（含旧格式固定IV）仍可验证；`LicenseDisplayInfo` 移除了内部字段 `EncryptSalt`，`LicenseID`/`LicenseQuantity` 已正式收进 `AppLicenseInfo`
3. **CLI变化**：`-k` 不再有默认弱密钥，必须显式传入；错误信息统一输出到 stderr；`genkeys` 默认拒绝覆盖已存在的密钥文件（需 `-force` 确认）
4. **错误判定**：验证失败的错误文案有调整（如"授权文件不适用于此设备"改为由 `ErrDeviceMismatch` 包装的新文案）；请勿对错误做字符串匹配，改用 `errors.Is` 判定哨兵错误（`ErrLicenseExpired`/`ErrDeviceMismatch`/`ErrInvalidSignature`/`ErrClockRollback`/`ErrUniversalRequiresExpiry` 等）
5. **行为变更提示**：v1/v2 签发入口（`EncryptLicToFile`/`EncryptLicSigned`/`EncryptLicSignedUniversal`）现要求 `LimitedTime` 非空时必须为合法 `YYYYMMDD` 日期（此前允许任意非空字符串）。验证端早已按此标准拒绝，该变更只把必然无效的签发拦在源头；存量调用方如传入过非法日期格式，会在签发时收到明确报错
6. **推荐迁移**：新签发许可证一律使用 v2 Ed25519 签名格式（`genkeys` + `-privkey`/`-pubkey`），客户端只需内嵌公钥

### 从v1.0.x分散工具升级

```bash
# v1.0.x → v1.1.0+
cmd/CreateLic/main.go     → licensemanager create
cmd/VerifyLic/main.go     → licensemanager verify  
cmd/ShowLicenseInfo/main.go → licensemanager show
cmd/checkuuid/main.go     → licensemanager checkuuid
```


## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可证

本项目使用MIT许可证。详见LICENSE文件。
