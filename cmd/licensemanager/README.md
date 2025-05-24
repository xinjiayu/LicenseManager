# LicenseManager 统一工具

LicenseManager 统一工具将原来分散的多个工具合并为一个，通过子命令来实现不同的功能，简化了工具的使用和维护。

## 安装

```bash
# 编译为可执行文件
go build -o licensemanager main.go

# 或者直接运行
go run main.go <command> [options]
```

## 命令概览

```
LicenseManager 1.1.0 - 统一的许可证管理工具

Usage: licensemanager <command> [options]

Commands:
  create      创建许可证文件
  verify      验证许可证文件
  show        显示许可证详细信息 (别名: info)
  checkuuid   获取当前设备的UUID (别名: uuid)
  help        显示帮助信息
  version     显示版本信息
```

## 详细使用说明

### 1. 创建许可证 (create)

从配置文件创建许可证文件。

```bash
# 基本用法
licensemanager create -f config.json -k "1234567890123456"

# 指定输出文件名
licensemanager create -f config.json -k "your-secret-key" -o my-license.lic

# 查看帮助
licensemanager create -h
```

**参数说明**：
- `-f`: 配置文件路径（必需）
- `-k`: 加密密钥，支持16/24/32字节长度（默认: "1234567890123456"）
- `-o`: 输出文件名（默认: "app.lic"）
- `-h`: 显示帮助信息

### 2. 验证许可证 (verify)

验证许可证文件的有效性。

```bash
# 基本用法
licensemanager verify -f app.lic -k "1234567890123456"

# 验证指定文件
licensemanager verify -f /path/to/license.lic -k "your-secret-key"

# 查看帮助
licensemanager verify -h
```

**参数说明**：
- `-f`: 许可证文件路径（默认: "app.lic"）
- `-k`: 解密密钥（默认: "1234567890123456"）
- `-h`: 显示帮助信息

### 3. 显示许可证信息 (show/info)

显示许可证文件的详细信息。

```bash
# 文本格式显示
licensemanager show -f app.lic -k "1234567890123456"

# JSON格式显示
licensemanager show -f app.lic -k "1234567890123456" -format json

# 使用别名
licensemanager info -f app.lic -format json

# 查看帮助
licensemanager show -h
```

**参数说明**：
- `-f`: 许可证文件路径（默认: "app.lic"）
- `-k`: 解密密钥（默认: "1234567890123456"）
- `-format`: 输出格式，支持 text/json（默认: "text"）
- `-h`: 显示帮助信息

### 4. 获取设备UUID (checkuuid/uuid)

获取当前设备的唯一标识符，用于生成许可证。

```bash
# 基本用法
licensemanager checkuuid

# 使用别名
licensemanager uuid

# 查看帮助
licensemanager checkuuid -h
```

### 5. 帮助和版本信息

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

### 完整工作流程

1. **获取设备UUID**：
```bash
$ licensemanager checkuuid
获取设备UUID...
设备UUID: F6235A40-C9E2-5681-B236-ED9C4C15E58D
```

2. **创建配置文件** (`config.json`)：
```json
{
  "AppName":"SagooIOT",
  "LicenseID": "LIC1234567890",
  "LicenseQuantity": 10,
  "AppCompany":"Shenyang Sagoo Education Technology Co., Ltd",
  "EncryptSalt":"afsrewfs2342423",
  "AppUUID":"55667788811111111111111111",
  "ObjUUID":"F6235A40-C9E2-5681-B236-ED9C4C15E58D",
  "AuthorizedName":"XXXXX公司",
  "LimitedTime":"20260601"
}
```

3. **生成许可证**：
```bash
$ licensemanager create -f config.json -k "my-secret-key-16b" -o production.lic
使用配置文件: config.json
密钥长度: 16字节
输出文件: production.lic
✅ 许可证文件生成成功！
```

4. **验证许可证**：
```bash
$ licensemanager verify -f production.lic -k "my-secret-key-16b"
验证许可证文件: production.lic
使用密钥长度: 16字节
✅ 许可证验证成功！
```

5. **查看许可证信息**：
```bash
$ licensemanager show -f production.lic -k "my-secret-key-16b"
读取许可证文件: production.lic
使用密钥长度: 16字节
--------------------------------------------------
=== 许可证信息 ===
应用名称: SagooIOT
发布公司: Shenyang Sagoo Education Technology Co., Ltd
授权名称: XXXXX公司
许可证ID: LIC1234567890
许可证数量: 10
目标设备UUID: F6235A40-C9E2-5681-B236-ED9C4C15E58D
应用UUID: 55667788811111111111111111
到期时间: 20260601
剩余天数: 372天
许可证状态: ✅ 有效
```

### JSON API 集成

获取JSON格式的许可证信息，便于Web API集成：

```bash
$ licensemanager show -f production.lic -k "my-secret-key-16b" -format json
{
  "app_name": "SagooIOT",
  "app_company": "Shenyang Sagoo Education Technology Co., Ltd",
  "app_uuid": "55667788811111111111111111",
  "obj_uuid": "F6235A40-C9E2-5681-B236-ED9C4C15E58D",
  "authorized_name": "XXXXX公司",
  "limited_time": "20260601",
  "license_id": "LIC1234567890",
  "license_quantity": 10,
  "encrypt_salt": "afsrewfs2342423",
  "status": "valid",
  "days_remaining": 372,
  "is_valid": true,
  "error_message": ""
}
```

## 错误处理

工具会提供清晰的错误信息：

```bash
# 缺少必需参数
$ licensemanager create
错误：必须指定配置文件 (-f)

# 无效密钥长度
$ licensemanager create -f config.json -k "123"
错误：密钥长度无效，当前长度为3字节，必须是16、24或32字节

# 文件不存在
$ licensemanager verify -f nonexistent.lic
❌ 许可证验证失败: 授权文件不存在
```

## 与原有工具的对应关系

| 原有工具 | 新统一工具命令 |
|---------|---------------|
| `cmd/CreateLic/main.go` | `licensemanager create` |
| `cmd/VerifyLic/main.go` | `licensemanager verify` |
| `cmd/ShowLicenseInfo/main.go` | `licensemanager show` |
| `cmd/checkuuid/main.go` | `licensemanager checkuuid` |

## 优势

1. **统一界面**：所有功能通过一个工具访问
2. **减少重复**：共享通用代码，减少维护成本  
3. **一致性**：统一的参数格式和错误处理
4. **易于分发**：只需分发一个可执行文件
5. **扩展性**：新功能作为子命令轻松添加 