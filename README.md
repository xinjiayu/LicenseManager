# LicenseManager v1.1.0 - 统一许可证管理工具

## 🎯 概述

LicenseManager 是一个功能完整的许可证管理系统，用于生成、验证和管理软件许可证。v1.1.0 版本将原来分散的多个工具合并为一个统一的命令行工具，提供了更好的用户体验和更强的功能。

## ✨ 主要特性

- 🔐 **AES加密**：支持16/24/32字节密钥，使用随机IV确保安全性
- 📱 **设备绑定**：基于设备UUID的许可证绑定机制
- ⏰ **到期管理**：支持许可证过期时间设置和剩余天数计算
- 🛠️ **统一工具**：一个命令行工具包含所有功能
- 📊 **多种输出**：支持文本和JSON格式输出
- 🔍 **详细信息**：完整的许可证信息查看功能

## 🚀 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/xinjiayu/LicenseManager.git
cd LicenseManager

# 编译统一工具
cd cmd/licensemanager
go build -o licensemanager main.go

# 或者直接运行
go run main.go help
```

### 基本使用

```bash
# 1. 获取设备UUID
./licensemanager checkuuid

# 2. 创建许可证
./licensemanager create -f config.json -k "your-secret-key"

# 3. 验证许可证
./licensemanager verify -f app.lic -k "your-secret-key"

# 4. 查看许可证信息
./licensemanager show -f app.lic -k "your-secret-key"
```

## 📖 详细功能

### 1. 创建许可证 (create)

从JSON配置文件生成加密的许可证文件。

```bash
# 基本用法
licensemanager create -f config.json -k "1234567890123456"

# 指定输出文件
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
- 文件完整性检查
- 设备UUID匹配验证
- 过期时间检查

```bash
licensemanager verify -f app.lic -k "your-secret-key"
```

### 3. 显示许可证信息 (show/info)

获取许可证的详细信息，支持文本和JSON格式。

```bash
# 文本格式（适合终端查看）
licensemanager show -f app.lic -k "your-secret-key"

# JSON格式（适合API集成）
licensemanager show -f app.lic -k "your-secret-key" -format json
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

## 🔧 API 使用

### Go API 集成

```go
package main

import (
    "fmt"
    "github.com/xinjiayu/LicenseManager"
)

func main() {
    // 获取许可证信息
    info, err := LicenseManager.GetLicenseInfo("app.lic", "your-secret-key")
    if err != nil {
        panic(err)
    }
    
    if info.IsValid {
        fmt.Printf("应用: %s, 剩余: %d天\n", info.AppName, info.DaysRemaining)
    } else {
        fmt.Printf("许可证无效: %s\n", info.ErrorMessage)
    }
    
    // 验证许可证
    isValid, err := LicenseManager.ValidAppLic("app.lic", "your-secret-key")
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

1. **AES加密**：使用工业标准AES加密算法
2. **随机IV**：每次加密使用不同的初始化向量
3. **设备绑定**：许可证与特定设备UUID绑定
4. **密钥验证**：支持16/24/32字节标准密钥长度
5. **完整性检查**：PKCS7填充确保数据完整性

## 📁 项目结构

```
LicenseManager/
├── cmd/
│   └── licensemanager/          # 统一命令行工具
│       ├── main.go             # 主程序入口
│       └── README.md           # 工具使用说明
├── utils/
│   └── ase.go                  # AES加密解密工具
├── examples/
│   └── license_info/           # 使用示例
├── lic.go                      # 核心许可证功能
├── lic_test.go                 # 单元测试
├── go.mod                      # Go模块定义
└── README.md           # 本文档
```

## 🧪 测试

```bash
# 运行单元测试
go test -v

# 测试完整工作流程
cd cmd/licensemanager

# 获取设备UUID
go run main.go checkuuid

# 创建测试许可证
go run main.go create -f ../../cmd/CreateLic/aaa.json

# 验证许可证
go run main.go verify -f app.lic

# 查看详细信息
go run main.go show -f app.lic -format json
```

## 🔄 迁移指南

### 从v1.0.x升级

v1.1.0与v1.0.x完全兼容：

1. **API兼容**：所有原有函数保持不变
2. **文件兼容**：支持旧版本许可证文件
3. **工具替换**：使用新的统一工具替代原来的分散工具

**工具命令对应关系**：
```bash
# v1.0.x → v1.1.0
cmd/CreateLic/main.go     → licensemanager create
cmd/VerifyLic/main.go     → licensemanager verify  
cmd/ShowLicenseInfo/main.go → licensemanager show
cmd/checkuuid/main.go     → licensemanager checkuuid
```


## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可证

本项目使用MIT许可证。详见LICENSE文件。
