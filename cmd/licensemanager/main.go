package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager"
)

const (
	Version = "1.2.0"
)

// fatalf 错误信息统一输出到stderr并退出
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
	os.Exit(1)
}

var (
	// 全局标志
	help    bool
	version bool
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// 处理全局选项
	if command == "-h" || command == "--help" || command == "help" {
		showUsage()
		return
	}

	if command == "-v" || command == "--version" || command == "version" {
		fmt.Printf("LicenseManager version %s\n", Version)
		return
	}

	// 执行子命令
	switch command {
	case "create":
		createCommand(os.Args[2:])
	case "verify":
		verifyCommand(os.Args[2:])
	case "show", "info":
		showCommand(os.Args[2:])
	case "checkuuid", "uuid":
		checkUUIDCommand(os.Args[2:])
	case "genkeys", "genkey":
		genKeysCommand(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "错误：未知命令 '%s'\n\n", command)
		showUsage()
		os.Exit(1)
	}
}

// readKeyFile 读取密钥PEM文件内容
func readKeyFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取密钥文件失败: %w", err)
	}
	return string(data), nil
}

// requireAESKey 校验v1模式必须显式提供AES密钥，且长度为16/24/32字节
func requireAESKey(key string) {
	if key == "" {
		fatalf("v1模式必须显式指定授权密钥 (-k)，且不建议继续使用v1格式，推荐改用 -privkey 生成v2签名许可证")
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		fatalf("密钥长度无效，当前长度为%d字节，必须是16、24或32字节", len(key))
	}
}

// genKeysCommand 生成 Ed25519 签名密钥对
func genKeysCommand(args []string) {
	fs := flag.NewFlagSet("genkeys", flag.ExitOnError)
	var (
		outputDir = fs.String("o", ".", "密钥文件输出目录")
		force     = fs.Bool("force", false, "覆盖已存在的密钥文件（危险：覆盖私钥后，旧密钥签发的许可证将无法续签同密钥新证）")
		help      = fs.Bool("h", false, "查看帮助信息")
	)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `genkeys - 生成Ed25519签名密钥对

私钥用于签发许可证（务必妥善保管），公钥内嵌到客户端用于验证。

Usage: licensemanager genkeys [options]

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  licensemanager genkeys
  licensemanager genkeys -o ./keys
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fatalf("创建输出目录失败: %v", err)
	}

	privatePath := filepath.Join(*outputDir, "license_private_key.pem")
	publicPath := filepath.Join(*outputDir, "license_public_key.pem")

	// 已有密钥文件时拒绝覆盖：覆盖私钥后若旧私钥无备份，已签发的许可证将无法续签同密钥新证
	if !*force {
		for _, p := range []string{privatePath, publicPath} {
			if _, err := os.Stat(p); err == nil {
				fatalf("密钥文件已存在: %s（确认要覆盖请显式指定 -force）", p)
			}
		}
	}

	privatePEM, publicPEM, err := LicenseManager.GenerateSigningKeyPair()
	if err != nil {
		fatalf("生成密钥对失败: %v", err)
	}

	// 私钥文件收紧权限，仅当前用户可读写
	if err := os.WriteFile(privatePath, []byte(privatePEM), 0600); err != nil {
		fatalf("写入私钥文件失败: %v", err)
	}
	if err := os.WriteFile(publicPath, []byte(publicPEM), 0644); err != nil {
		fatalf("写入公钥文件失败: %v", err)
	}

	fmt.Println("✅ Ed25519 密钥对生成成功！")
	fmt.Printf("私钥文件（签发许可证用，请妥善保管）: %s\n", privatePath)
	fmt.Printf("公钥文件（内嵌到客户端用于验证）: %s\n", publicPath)
}

// createCommand 创建许可证
// 默认使用 v1 AES 对称加密格式；指定 -privkey 时使用 v2 Ed25519 签名格式（推荐）
func createCommand(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	var (
		configFile = fs.String("f", "", "需要授权的应用信息配置文件，json格式")
		key        = fs.String("k", "", "v1模式的AES授权密钥，16/24/32字节（已不推荐，建议改用-privkey）")
		privateKey = fs.String("privkey", "", "Ed25519私钥PEM文件路径，指定后生成v2签名格式许可证（推荐）")
		output     = fs.String("o", "app.lic", "输出的许可证文件名")
		help       = fs.Bool("h", false, "查看帮助信息")
	)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `create - 创建许可证文件

Usage: licensemanager create [options]

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # v2 Ed25519签名格式（推荐，客户端只需公钥即可验证）
  licensemanager create -f config.json -privkey license_private_key.pem -o app.lic

  # v1 AES加密格式（不推荐，仅兼容旧版本客户端）
  licensemanager create -f config.json -k "your-16-byte-key" -o app.lic
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	if *configFile == "" {
		fatalf("必须指定配置文件 (-f)")
	}

	fmt.Printf("使用配置文件: %s\n", *configFile)
	fmt.Printf("输出文件: %s\n", *output)

	if *privateKey != "" {
		// v2 签名模式
		if *key != "" {
			fmt.Fprintln(os.Stderr, "⚠️  同时指定了 -k 与 -privkey，将生成 v2 签名格式许可证，-k 被忽略")
		}
		privatePEM, err := readKeyFile(*privateKey)
		if err != nil {
			fatalf("%v", err)
		}

		fmt.Printf("签名模式: Ed25519 (v2)\n")
		if err := LicenseManager.EncryptLicSigned(*configFile, *output, privatePEM); err != nil {
			fatalf("创建许可证失败: %v", err)
		}
		fmt.Println("✅ 许可证文件生成成功！")
		return
	}

	// v1 AES 加密模式
	requireAESKey(*key)

	fmt.Printf("加密模式: AES (v1)\n")
	fmt.Printf("密钥长度: %d字节\n", len(*key))
	fmt.Println("⚠️  v1 AES格式已不推荐：客户端内嵌的解密密钥可被用于伪造许可证，建议改用 -privkey 生成v2签名格式")

	if err := LicenseManager.EncryptLicToFile(*configFile, *output, *key); err != nil {
		fatalf("创建许可证失败: %v", err)
	}

	fmt.Println("✅ 许可证文件生成成功！")
}

// verifyCommand 验证许可证
// 默认验证 v1 AES 格式；指定 -pubkey 时验证 v2 Ed25519 签名格式
func verifyCommand(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	var (
		licenseFile = fs.String("f", "app.lic", "要验证的许可证文件路径")
		key         = fs.String("k", "", "v1模式的AES解密密钥（已不推荐，建议改用-pubkey）")
		publicKey   = fs.String("pubkey", "", "Ed25519公钥PEM文件路径，指定后验证v2签名格式许可证")
		help        = fs.Bool("h", false, "查看帮助信息")
	)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `verify - 验证许可证文件

Usage: licensemanager verify [options]

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  licensemanager verify -f app.lic -pubkey license_public_key.pem
  licensemanager verify -f app.lic -k "your-16-byte-key"
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	fmt.Printf("验证许可证文件: %s\n", *licenseFile)

	var isValid bool
	var err error

	if *publicKey != "" {
		if *key != "" {
			fmt.Fprintln(os.Stderr, "⚠️  同时指定了 -k 与 -pubkey，将按 v2 签名格式验证，-k 被忽略")
		}
		publicPEM, keyErr := readKeyFile(*publicKey)
		if keyErr != nil {
			fatalf("%v", keyErr)
		}
		fmt.Printf("验证模式: Ed25519签名 (v2)\n")
		isValid, err = LicenseManager.ValidAppLicSigned(*licenseFile, publicPEM)
	} else {
		requireAESKey(*key)
		fmt.Printf("验证模式: AES解密 (v1)\n")
		fmt.Printf("使用密钥长度: %d字节\n", len(*key))
		isValid, err = LicenseManager.ValidAppLic(*licenseFile, *key)
	}

	if err != nil {
		fatalf("许可证验证失败: %s", err.Error())
	}

	if isValid {
		fmt.Println("✅ 许可证验证成功！")
	} else {
		fatalf("许可证验证失败！")
	}
}

// showCommand 显示许可证信息
// 默认读取 v1 AES 格式；指定 -pubkey 时读取 v2 Ed25519 签名格式
func showCommand(args []string) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	var (
		licenseFile = fs.String("f", "app.lic", "要读取的许可证文件路径")
		key         = fs.String("k", "", "v1模式的AES解密密钥（已不推荐，建议改用-pubkey）")
		publicKey   = fs.String("pubkey", "", "Ed25519公钥PEM文件路径，指定后读取v2签名格式许可证")
		format      = fs.String("format", "text", "输出格式：text（文本）或 json（JSON）")
		help        = fs.Bool("h", false, "查看帮助信息")
	)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `show - 显示许可证详细信息

Usage: licensemanager show [options]

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  licensemanager show -f app.lic -pubkey license_public_key.pem
  licensemanager show -f app.lic -pubkey license_public_key.pem -format json
  licensemanager show -f app.lic -k "your-16-byte-key"
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	if *publicKey == "" && *key == "" {
		fatalf("必须指定 -pubkey（v2签名格式）或 -k（v1 AES格式）其中一个")
	}
	if *publicKey != "" && *key != "" {
		fmt.Fprintln(os.Stderr, "⚠️  同时指定了 -k 与 -pubkey，将按 v2 签名格式读取，-k 被忽略")
	}

	fmt.Printf("读取许可证文件: %s\n", *licenseFile)
	fmt.Println(strings.Repeat("-", 50))

	switch *format {
	case "json":
		showJSONFormat(*licenseFile, *key, *publicKey)
	case "text":
		showTextFormat(*licenseFile, *key, *publicKey)
	default:
		fatalf("不支持的输出格式 '%s'，支持的格式：text, json", *format)
	}
}

// checkUUIDCommand 检查设备UUID
func checkUUIDCommand(args []string) {
	fs := flag.NewFlagSet("checkuuid", flag.ExitOnError)
	var (
		help = fs.Bool("h", false, "查看帮助信息")
	)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `checkuuid - 获取当前设备的UUID

Usage: licensemanager checkuuid [options]

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  licensemanager checkuuid
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	fmt.Println("获取设备UUID...")
	id, err := machineid.ID()
	if err != nil {
		fatalf("获取设备UUID失败: %v", err)
	}

	fmt.Printf("设备UUID: %s\n", id)
}

// showTextFormat 显示文本格式
func showTextFormat(licenseFile, key, publicKeyFile string) {
	var formatted string
	var err error

	if publicKeyFile != "" {
		publicPEM, keyErr := readKeyFile(publicKeyFile)
		if keyErr != nil {
			fatalf("%v", keyErr)
		}
		formatted, err = LicenseManager.GetLicenseInfoSignedFormatted(licenseFile, publicPEM)
	} else {
		formatted, err = LicenseManager.GetLicenseInfoFormatted(licenseFile, key)
	}

	if err != nil {
		fatalf("读取许可证信息失败: %s", err.Error())
	}

	fmt.Println(formatted)
}

// showJSONFormat 显示JSON格式
func showJSONFormat(licenseFile, key, publicKeyFile string) {
	var info *LicenseManager.LicenseDisplayInfo
	var err error

	if publicKeyFile != "" {
		publicPEM, keyErr := readKeyFile(publicKeyFile)
		if keyErr != nil {
			fatalf("%v", keyErr)
		}
		info, err = LicenseManager.GetLicenseInfoSigned(licenseFile, publicPEM)
	} else {
		info, err = LicenseManager.GetLicenseInfo(licenseFile, key)
	}

	if err != nil {
		fatalf("读取许可证信息失败: %s", err.Error())
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fatalf("JSON序列化失败: %s", err.Error())
	}

	fmt.Println(string(jsonData))
}

// showUsage 显示主要使用说明
func showUsage() {
	fmt.Fprintf(os.Stderr, `LicenseManager %s - 统一的许可证管理工具

Usage: licensemanager <command> [options]

Commands:
  create      创建许可证文件（-privkey 生成v2签名格式，推荐）
  verify      验证许可证文件（-pubkey 验证v2签名格式）
  show        显示许可证详细信息 (别名: info，支持 -pubkey)
  checkuuid   获取当前设备的UUID (别名: uuid)
  genkeys     生成Ed25519签名密钥对 (别名: genkey)
  help        显示帮助信息
  version     显示版本信息

Global Options:
  -h, --help     显示帮助信息
  -v, --version  显示版本信息

Examples:
  licensemanager genkeys -o ./keys
  licensemanager create -f config.json -privkey keys/license_private_key.pem
  licensemanager verify -f app.lic -pubkey keys/license_public_key.pem
  licensemanager show -f app.lic -pubkey keys/license_public_key.pem -format json
  licensemanager checkuuid

Use "licensemanager <command> -h" for more information about a command.
`, Version)
}
