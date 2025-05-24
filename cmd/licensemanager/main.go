package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager"
)

const (
	Version = "1.1.0"
)

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
	default:
		fmt.Fprintf(os.Stderr, "错误：未知命令 '%s'\n\n", command)
		showUsage()
		os.Exit(1)
	}
}

// createCommand 创建许可证
func createCommand(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	var (
		configFile = fs.String("f", "", "需要授权的应用信息配置文件，json格式")
		key        = fs.String("k", "1234567890123456", "授权密钥（支持16、24或32字节长度）")
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
  licensemanager create -f config.json -k "1234567890123456" -o app.lic
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	if *configFile == "" {
		fmt.Fprintf(os.Stderr, "错误：必须指定配置文件 (-f)\n")
		fs.Usage()
		os.Exit(1)
	}

	if *key == "" {
		fmt.Fprintf(os.Stderr, "错误：必须指定授权密钥 (-k)\n")
		fs.Usage()
		os.Exit(1)
	}

	// 验证密钥长度：AES支持16、24、32字节的密钥
	keyLen := utf8.RuneCountInString(*key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		fmt.Fprintf(os.Stderr, "错误：密钥长度无效，当前长度为%d字节，必须是16、24或32字节\n", keyLen)
		os.Exit(1)
	}

	fmt.Printf("使用配置文件: %s\n", *configFile)
	fmt.Printf("密钥长度: %d字节\n", keyLen)
	fmt.Printf("输出文件: %s\n", *output)

	// 临时修改输出文件名，因为 EncryptLic 固定输出 app.lic
	originalDir, _ := os.Getwd()
	LicenseManager.EncryptLic(*configFile, *key)

	// 如果指定了不同的输出文件名，重命名
	if *output != "app.lic" {
		if err := os.Rename("app.lic", *output); err != nil {
			fmt.Printf("警告：重命名文件失败: %v\n", err)
		} else {
			fmt.Printf("许可证文件已重命名为: %s\n", *output)
		}
	}

	fmt.Println("✅ 许可证文件生成成功！")

	_ = originalDir // 避免未使用变量警告
}

// verifyCommand 验证许可证
func verifyCommand(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	var (
		licenseFile = fs.String("f", "app.lic", "要验证的许可证文件路径")
		key         = fs.String("k", "1234567890123456", "解密密钥")
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
  licensemanager verify -f app.lic -k "1234567890123456"
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	fmt.Printf("验证许可证文件: %s\n", *licenseFile)
	fmt.Printf("使用密钥长度: %d字节\n", len(*key))

	isValid, err := LicenseManager.ValidAppLic(*licenseFile, *key)
	if err != nil {
		fmt.Printf("❌ 许可证验证失败: %s\n", err.Error())
		os.Exit(1)
	}

	if isValid {
		fmt.Println("✅ 许可证验证成功！")
	} else {
		fmt.Println("❌ 许可证验证失败！")
		os.Exit(1)
	}
}

// showCommand 显示许可证信息
func showCommand(args []string) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	var (
		licenseFile = fs.String("f", "app.lic", "要读取的许可证文件路径")
		key         = fs.String("k", "1234567890123456", "解密密钥")
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
  licensemanager show -f app.lic -k "1234567890123456"
  licensemanager show -f app.lic -k "1234567890123456" -format json
`)
	}

	fs.Parse(args)

	if *help {
		fs.Usage()
		return
	}

	fmt.Printf("读取许可证文件: %s\n", *licenseFile)
	fmt.Printf("使用密钥长度: %d字节\n", len(*key))
	fmt.Println(strings.Repeat("-", 50))

	switch *format {
	case "json":
		showJSONFormat(*licenseFile, *key)
	case "text":
		showTextFormat(*licenseFile, *key)
	default:
		fmt.Fprintf(os.Stderr, "错误：不支持的输出格式 '%s'，支持的格式：text, json\n", *format)
		os.Exit(1)
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
		fmt.Printf("❌ 获取设备UUID失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("设备UUID: %s\n", id)
}

// showTextFormat 显示文本格式
func showTextFormat(licenseFile, key string) {
	formatted, err := LicenseManager.GetLicenseInfoFormatted(licenseFile, key)
	if err != nil {
		fmt.Printf("❌ 读取许可证信息失败: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println(formatted)
}

// showJSONFormat 显示JSON格式
func showJSONFormat(licenseFile, key string) {
	info, err := LicenseManager.GetLicenseInfo(licenseFile, key)
	if err != nil {
		fmt.Printf("❌ 读取许可证信息失败: %s\n", err.Error())
		os.Exit(1)
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

// showUsage 显示主要使用说明
func showUsage() {
	fmt.Fprintf(os.Stderr, `LicenseManager %s - 统一的许可证管理工具

Usage: licensemanager <command> [options]

Commands:
  create      创建许可证文件
  verify      验证许可证文件
  show        显示许可证详细信息 (别名: info)
  checkuuid   获取当前设备的UUID (别名: uuid)
  help        显示帮助信息
  version     显示版本信息

Global Options:
  -h, --help     显示帮助信息
  -v, --version  显示版本信息

Examples:
  licensemanager create -f config.json -k "1234567890123456"
  licensemanager verify -f app.lic -k "1234567890123456"
  licensemanager show -f app.lic -k "1234567890123456" -format json
  licensemanager checkuuid

Use "licensemanager <command> -h" for more information about a command.
`, Version)
}
