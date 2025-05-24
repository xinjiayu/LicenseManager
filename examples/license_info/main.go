package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xinjiayu/LicenseManager"
)

func main() {
	// 示例1：获取结构化的许可证信息
	fmt.Println("=== 示例1：获取结构化许可证信息 ===")
	licenseFile := "../../cmd/CreateLic/app.lic"
	key := "1234567890123456"

	info, err := LicenseManager.GetLicenseInfo(licenseFile, key)
	if err != nil {
		log.Fatalf("获取许可证信息失败: %v", err)
	}

	// 检查许可证状态
	if info.IsValid {
		fmt.Printf("✅ 许可证有效！应用: %s\n", info.AppName)
		fmt.Printf("授权公司: %s\n", info.AuthorizedName)
		if info.DaysRemaining >= 0 {
			fmt.Printf("剩余天数: %d天\n", info.DaysRemaining)
		} else {
			fmt.Println("永久许可证")
		}
	} else {
		fmt.Printf("❌ 许可证无效: %s\n", info.ErrorMessage)
	}

	fmt.Println()

	// 示例2：获取格式化的文本信息
	fmt.Println("=== 示例2：获取格式化文本信息 ===")
	formatted, err := LicenseManager.GetLicenseInfoFormatted(licenseFile, key)
	if err != nil {
		log.Fatalf("获取格式化信息失败: %v", err)
	}
	fmt.Print(formatted)

	fmt.Println()

	// 示例3：将许可证信息转换为JSON（用于Web API）
	fmt.Println("=== 示例3：JSON格式输出（适用于Web API）===")
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatalf("JSON序列化失败: %v", err)
	}
	fmt.Println(string(jsonData))

	fmt.Println()

	// 示例4：在应用中使用许可证验证
	fmt.Println("=== 示例4：应用启动时的许可证检查 ===")
	if !checkLicenseForApp(licenseFile, key, "SagooIOT") {
		fmt.Println("应用启动失败：许可证验证不通过")
		return
	}
	fmt.Println("应用启动成功：许可证验证通过")
}

// checkLicenseForApp 检查特定应用的许可证
func checkLicenseForApp(licenseFile, key, expectedAppName string) bool {
	info, err := LicenseManager.GetLicenseInfo(licenseFile, key)
	if err != nil {
		fmt.Printf("许可证检查失败: %v\n", err)
		return false
	}

	// 检查许可证是否有效
	if !info.IsValid {
		fmt.Printf("许可证无效: %s\n", info.ErrorMessage)
		return false
	}

	// 检查应用名称是否匹配
	if info.AppName != expectedAppName {
		fmt.Printf("许可证应用名称不匹配: 期望 %s, 实际 %s\n", expectedAppName, info.AppName)
		return false
	}

	// 检查是否即将过期（少于30天）
	if info.DaysRemaining >= 0 && info.DaysRemaining <= 30 {
		fmt.Printf("⚠️  警告：许可证将在 %d 天后过期，请及时续费\n", info.DaysRemaining)
	}

	return true
}
