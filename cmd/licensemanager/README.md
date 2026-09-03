# LicenseManager 统一工具

LicenseManager 统一工具将原来分散的多个工具合并为一个，通过子命令来实现不同的功能，简化了工具的使用和维护。

## 安装

```bash
# 在仓库根目录编译为可执行文件
go build -o licensemanager ./cmd/licensemanager

# 或者直接运行
go run ./cmd/licensemanager <command> [options]
```

## 命令概览

```
LicenseManager 1.2.0 - 统一的许可证管理工具

Usage: licensemanager <command> [options]

Commands:
  create      创建许可证文件（-privkey 生成v2签名格式，推荐）
  verify      验证许可证文件（-pubkey 验证v2签名格式）
  show        显示许可证详细信息 (别名: info，支持 -pubkey)
  checkuuid   获取当前设备的UUID (别名: uuid)
  genkeys     生成Ed25519签名密钥对 (别名: genkey)
  help        显示帮助信息
  version     显示版本信息
```

## 许可证格式选择

- **v2 Ed25519 签名（推荐）**：签发方持有私钥，客户端只需内嵌公钥即可验证，无法伪造许可证
- **v1 AES 对称加密（已不推荐）**：仅用于兼容存量许可证；客户端内嵌的解密密钥可被用于伪造许可证

## 详细使用说明

### 1. 生成签名密钥对 (genkeys/genkey)

生成 Ed25519 密钥对，私钥用于签发许可证（务必妥善保管），公钥内嵌到客户端用于验证。

```bash
# 基本用法（输出到当前目录）
licensemanager genkeys

# 指定输出目录
licensemanager genkeys -o ./keys

# 覆盖已存在的密钥文件（危险操作，需显式确认）
licensemanager genkeys -o ./keys -force

# 查看帮助
licensemanager genkeys -h
```

**参数说明**：
- `-o`: 密钥文件输出目录（默认: "."）
- `-force`: 覆盖已存在的密钥文件（默认拒绝覆盖，防止误操作导致旧私钥无法找回）
- `-h`: 显示帮助信息

**输出文件**：
- `license_private_key.pem`：私钥（权限 0600，签发许可证用）
- `license_public_key.pem`：公钥（权限 0644，内嵌到客户端验证）

### 2. 创建许可证 (create)

从配置文件创建许可证文件。

```bash
# v2 Ed25519签名格式（推荐）
licensemanager create -f config.json -privkey keys/license_private_key.pem -o app.lic

# v1 AES加密格式（不推荐，仅兼容旧版本客户端；-k 必须显式传入）
licensemanager create -f config.json -k "your-16-byte-key" -o my-license.lic

# 查看帮助
licensemanager create -h
```

**参数说明**：
- `-f`: 配置文件路径（必需）
- `-privkey`: Ed25519 私钥 PEM 文件路径，指定后生成 v2 签名格式许可证（推荐）
- `-k`: v1 模式的 AES 密钥，16/24/32 字节（无默认值，必须显式传入）
- `-o`: 输出文件名（默认: "app.lic"）
- `-h`: 显示帮助信息

> 同时指定 `-k` 与 `-privkey` 时将使用 v2 签名格式，`-k` 被忽略（会输出提示）。

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

### 3. 验证许可证 (verify)

验证许可证文件的有效性，包括：签名验证（v2）或解密校验（v1）、设备UUID匹配、过期时间检查（按UTC）、系统时间回拨检测。

```bash
# v2 签名格式（推荐）
licensemanager verify -f app.lic -pubkey keys/license_public_key.pem

# v1 AES格式（不推荐）
licensemanager verify -f /path/to/license.lic -k "your-16-byte-key"

# 查看帮助
licensemanager verify -h
```

**参数说明**：
- `-f`: 许可证文件路径（默认: "app.lic"）
- `-pubkey`: Ed25519 公钥 PEM 文件路径，指定后验证 v2 签名格式许可证
- `-k`: v1 模式的 AES 解密密钥（无默认值，必须显式传入）
- `-h`: 显示帮助信息

> 同时指定 `-k` 与 `-pubkey` 时将按 v2 签名格式验证，`-k` 被忽略（会输出提示）。

### 4. 显示许可证信息 (show/info)

显示许可证文件的详细信息。

```bash
# v2 签名格式，文本显示（推荐）
licensemanager show -f app.lic -pubkey keys/license_public_key.pem

# JSON格式显示（适合API集成）
licensemanager show -f app.lic -pubkey keys/license_public_key.pem -format json

# v1 AES格式
licensemanager show -f app.lic -k "your-16-byte-key"

# 使用别名
licensemanager info -f app.lic -pubkey keys/license_public_key.pem

# 查看帮助
licensemanager show -h
```

**参数说明**：
- `-f`: 许可证文件路径（默认: "app.lic"）
- `-pubkey`: Ed25519 公钥 PEM 文件路径，指定后读取 v2 签名格式许可证
- `-k`: v1 模式的 AES 解密密钥（无默认值）
- `-format`: 输出格式，支持 text/json（默认: "text"）
- `-h`: 显示帮助信息

### 5. 获取设备UUID (checkuuid/uuid)

获取当前设备的唯一标识符，用于生成设备绑定的许可证。

```bash
# 基本用法
licensemanager checkuuid

# 使用别名
licensemanager uuid
```

### 6. 帮助和版本信息

```bash
# 显示主帮助
licensemanager help
licensemanager -h
licensemanager --help

# 显示版本信息
licensemanager version
licensemanager -v
licensemanager --version

# 显示子命令帮助
licensemanager <command> -h
```

## 实际使用示例

### 完整工作流程（v2 签名格式，推荐）

1. **生成密钥对**（签发方一次性操作）：
```bash
$ licensemanager genkeys -o ./keys
✅ Ed25519 密钥对生成成功！
私钥文件（签发许可证用，请妥善保管）: keys/license_private_key.pem
公钥文件（内嵌到客户端用于验证）: keys/license_public_key.pem
```

2. **获取设备UUID**（客户设备上执行，发给签发方）：
```bash
$ licensemanager checkuuid
获取设备UUID...
设备UUID: F6235A40-C9E2-5681-B236-ED9C4C15E58D
```

3. **创建配置文件** (`config.json`)：
```json
{
  "AppName":"SagooIOT",
  "LicenseID": "LIC1234567890",
  "LicenseQuantity": 10,
  "AppCompany":"Shenyang Sagoo Education Technology Co., Ltd",
  "AppUUID":"55667788811111111111111111",
  "ObjUUID":"F6235A40-C9E2-5681-B236-ED9C4C15E58D",
  "AuthorizedName":"XXXXX公司",
  "LimitedTime":"20261231"
}
```

4. **签发许可证**（签发方持有私钥）：
```bash
$ licensemanager create -f config.json -privkey keys/license_private_key.pem -o production.lic
使用配置文件: config.json
输出文件: production.lic
签名模式: Ed25519 (v2)
✅ 许可证文件生成成功！
```

5. **验证许可证**（客户端只需公钥）：
```bash
$ licensemanager verify -f production.lic -pubkey keys/license_public_key.pem
验证许可证文件: production.lic
验证模式: Ed25519签名 (v2)
✅ 许可证验证成功！
```

6. **查看许可证信息**：
```bash
$ licensemanager show -f production.lic -pubkey keys/license_public_key.pem
读取许可证文件: production.lic
--------------------------------------------------
=== 许可证信息 ===
应用名称: SagooIOT
发布公司: Shenyang Sagoo Education Technology Co., Ltd
授权名称: XXXXX公司
许可证ID: LIC1234567890
许可证数量: 10
目标设备UUID: F6235A40-C9E2-5681-B236-ED9C4C15E58D
应用UUID: 55667788811111111111111111
到期时间: 20261231
剩余天数: 118天
许可证状态: ✅ 有效
```

### JSON API 集成

获取JSON格式的许可证信息，便于Web API集成：

```bash
$ licensemanager show -f production.lic -pubkey keys/license_public_key.pem -format json
```

输出（前面的横幅行为提示信息，JSON 主体从 `{` 开始）：
```json
{
  "app_name": "SagooIOT",
  "app_company": "Shenyang Sagoo Education Technology Co., Ltd",
  "app_uuid": "55667788811111111111111111",
  "obj_uuid": "F6235A40-C9E2-5681-B236-ED9C4C15E58D",
  "authorized_name": "XXXXX公司",
  "limited_time": "20261231",
  "license_id": "LIC1234567890",
  "license_quantity": 10,
  "status": "valid",
  "days_remaining": 118,
  "is_valid": true,
  "error_message": ""
}
```

### v1 AES 格式（兼容存量许可证）

```bash
licensemanager create -f config.json -k "my-16-byte-key!!" -o legacy.lic
licensemanager verify -f legacy.lic -k "my-16-byte-key!!"
```

> ⚠️ v1 格式下客户端内嵌的解密密钥可被用于伪造许可证，新签发请一律使用 v2。

## 错误处理

工具会提供清晰的错误信息（统一输出到 stderr，退出码为 1）：

```bash
# 缺少必需参数
$ licensemanager create
❌ 必须指定配置文件 (-f)

# v1模式未显式指定密钥
$ licensemanager verify -f app.lic
❌ v1模式必须显式指定授权密钥 (-k)，且不建议继续使用v1格式，推荐改用 -privkey 生成v2签名许可证

# 无效密钥长度
$ licensemanager create -f config.json -k "123"
❌ 密钥长度无效，当前长度为3字节，必须是16、24或32字节

# 文件不存在
$ licensemanager verify -f nonexistent.lic -pubkey keys/license_public_key.pem
❌ 许可证验证失败: 授权文件不存在

# 密钥文件已存在，拒绝覆盖
$ licensemanager genkeys -o ./keys
❌ 密钥文件已存在: keys/license_private_key.pem（确认要覆盖请显式指定 -force）
```

## 与原有工具的对应关系

| 原有工具 | 新统一工具命令 |
|---------|---------------|
| `cmd/CreateLic/main.go` | `licensemanager create` |
| `cmd/VerifyLic/main.go` | `licensemanager verify` |
| `cmd/ShowLicenseInfo/main.go` | `licensemanager show` |
| `cmd/checkuuid/main.go` | `licensemanager checkuuid` |
| （新增） | `licensemanager genkeys` |

## 优势

1. **统一界面**：所有功能通过一个工具访问
2. **减少重复**：共享通用代码，减少维护成本  
3. **一致性**：统一的参数格式和错误处理
4. **易于分发**：只需分发一个可执行文件
5. **扩展性**：新功能作为子命令轻松添加 
