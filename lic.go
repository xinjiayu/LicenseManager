package LicenseManager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager/utils"
)

type AppLicenseInfo struct {
	AppName        string //应用名称
	AppCompany     string //应用发布的公司
	AppUUID        string //此次发布应用的UUID
	ObjUUID        string //目标设备的UUID
	AuthorizedName string //授权名称
	LimitedTime    string //到期日期
}

// LicenseDisplayInfo 用于显示的许可证信息结构
type LicenseDisplayInfo struct {
	AppName         string `json:"app_name"`         // 应用名称
	AppCompany      string `json:"app_company"`      // 应用发布公司
	AppUUID         string `json:"app_uuid"`         // 应用UUID
	ObjUUID         string `json:"obj_uuid"`         // 目标设备UUID
	AuthorizedName  string `json:"authorized_name"`  // 授权名称
	LimitedTime     string `json:"limited_time"`     // 到期日期
	LicenseID       string `json:"license_id"`       // 许可证ID（如果存在）
	LicenseQuantity int    `json:"license_quantity"` // 许可证数量（如果存在）
	EncryptSalt     string `json:"encrypt_salt"`     // 加密盐值（如果存在）
	Status          string `json:"status"`           // 许可证状态：valid, expired, invalid
	DaysRemaining   int    `json:"days_remaining"`   // 剩余天数
	IsValid         bool   `json:"is_valid"`         // 是否有效
	ErrorMessage    string `json:"error_message"`    // 错误信息（如果有）
}

// EncryptLic 根据应用信息的配置文件生成license授权文件
func EncryptLic(appInfoFile, key string) {
	// 从文件中读取配置
	file, err := os.Open(appInfoFile) // 使用Open而不是OpenFile，更简洁
	if err != nil {
		log.Fatalf("打开配置文件失败: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("关闭文件失败: %v", closeErr)
		}
	}()

	// 使用io.ReadAll替代已废弃的ioutil.ReadAll
	contentByte, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	// 验证JSON格式
	var conf AppLicenseInfo
	if err := json.Unmarshal(contentByte, &conf); err != nil {
		log.Fatalf("解析配置文件失败，请检查JSON格式: %v", err)
	}

	// 验证必要字段
	if conf.AppName == "" {
		log.Fatal("配置文件中AppName字段不能为空")
	}
	if conf.ObjUUID == "" {
		log.Fatal("配置文件中ObjUUID字段不能为空")
	}

	log.Printf("应用名称: %s", conf.AppName)
	log.Printf("目标设备UUID: %s", conf.ObjUUID)
	log.Printf("授权到期时间: %s", conf.LimitedTime)

	// 进行加密
	tmpText := string(contentByte)
	encryptedText := utils.AesEncrypt(tmpText, key)

	// 生成license授权文件
	licFilePath := "app.lic"
	dstFile, err := os.Create(licFilePath)
	if err != nil {
		log.Fatalf("创建授权文件失败: %v", err)
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil {
			log.Printf("关闭授权文件失败: %v", closeErr)
		}
	}()

	if _, err := dstFile.WriteString(encryptedText); err != nil {
		log.Fatalf("写入授权文件失败: %v", err)
	}

	log.Printf("授权文件已生成: %s", licFilePath)
}

// ValidAppLic 验证应用许可证
func ValidAppLic(appInfoFile, key string) (res bool, err error) {
	// 安全地打开文件
	file, err := os.Open(appInfoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.New("授权文件不存在")
		}
		return false, fmt.Errorf("打开授权文件失败: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("关闭授权文件失败: %v", closeErr)
		}
	}()

	// 读取文件内容
	contentByte, err := io.ReadAll(file)
	if err != nil {
		return false, errors.New("授权文件读取失败")
	}

	// 检查文件是否为空
	if len(contentByte) == 0 {
		return false, errors.New("授权文件为空")
	}

	// 进行解密
	tmpText := string(contentByte)
	decryptedText := utils.AesDecrypt(tmpText, key)

	// 解析JSON
	var conf AppLicenseInfo
	if err := json.Unmarshal([]byte(decryptedText), &conf); err != nil {
		return false, errors.New("授权文件格式错误或解密失败")
	}

	// 获取本机的ID
	id, err := machineid.ID()
	if err != nil {
		return false, errors.New("获取本机ID失败")
	}

	// 验证设备UUID
	if conf.ObjUUID != id {
		return false, errors.New("授权文件不适用于此设备")
	}

	// 验证到期时间
	limitedTime := conf.LimitedTime
	if limitedTime != "" {
		licDate, err := strconv.Atoi(limitedTime)
		if err != nil {
			return false, errors.New("授权文件中的到期时间格式错误")
		}

		nowDate := time.Now().Format("20060102")
		currentDate, err := strconv.Atoi(nowDate)
		if err != nil {
			return false, errors.New("系统时间格式错误")
		}

		if licDate < currentDate {
			errInfo := fmt.Sprintf("授权文件已过期！授权结束日期: %d, 当前日期: %d", licDate, currentDate)
			return false, errors.New(errInfo)
		}
	}

	return true, nil
}

// GetLicenseInfo 获取许可证文件的详细信息，用于界面显示
func GetLicenseInfo(licenseFile, key string) (*LicenseDisplayInfo, error) {
	// 初始化返回结构
	info := &LicenseDisplayInfo{
		Status:        "invalid",
		IsValid:       false,
		DaysRemaining: -1,
	}

	// 安全地打开文件
	file, err := os.Open(licenseFile)
	if err != nil {
		if os.IsNotExist(err) {
			info.ErrorMessage = "许可证文件不存在"
			return info, nil // 返回结构但不返回错误，让调用者决定如何处理
		}
		info.ErrorMessage = fmt.Sprintf("打开许可证文件失败: %v", err)
		return info, nil
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("关闭许可证文件失败: %v", closeErr)
		}
	}()

	// 读取文件内容
	contentByte, err := io.ReadAll(file)
	if err != nil {
		info.ErrorMessage = "许可证文件读取失败"
		return info, nil
	}

	// 检查文件是否为空
	if len(contentByte) == 0 {
		info.ErrorMessage = "许可证文件为空"
		return info, nil
	}

	// 尝试解密
	tmpText := string(contentByte)
	var decryptedText string

	// 使用recover来捕获解密过程中的panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				info.ErrorMessage = "许可证文件解密失败，可能密钥错误或文件损坏"
			}
		}()
		decryptedText = utils.AesDecrypt(tmpText, key)
	}()

	if info.ErrorMessage != "" {
		return info, nil
	}

	// 解析JSON到原始结构
	var conf AppLicenseInfo
	if err := json.Unmarshal([]byte(decryptedText), &conf); err != nil {
		info.ErrorMessage = "许可证文件格式错误"
		return info, nil
	}

	// 填充基本信息
	info.AppName = conf.AppName
	info.AppCompany = conf.AppCompany
	info.AppUUID = conf.AppUUID
	info.ObjUUID = conf.ObjUUID
	info.AuthorizedName = conf.AuthorizedName
	info.LimitedTime = conf.LimitedTime

	// 尝试解析额外字段（可能存在于JSON中但不在AppLicenseInfo结构中）
	var extraData map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedText), &extraData); err == nil {
		if licenseID, ok := extraData["LicenseID"].(string); ok {
			info.LicenseID = licenseID
		}
		if quantity, ok := extraData["LicenseQuantity"].(float64); ok {
			info.LicenseQuantity = int(quantity)
		}
		if salt, ok := extraData["EncryptSalt"].(string); ok {
			info.EncryptSalt = salt
		}
	}

	// 验证设备UUID
	id, err := machineid.ID()
	if err != nil {
		info.ErrorMessage = "获取本机ID失败"
		return info, nil
	}

	if conf.ObjUUID != id {
		info.Status = "invalid"
		info.ErrorMessage = "许可证文件不适用于此设备"
		return info, nil
	}

	// 验证到期时间
	if conf.LimitedTime != "" {
		licDate, err := strconv.Atoi(conf.LimitedTime)
		if err != nil {
			info.ErrorMessage = "许可证文件中的到期时间格式错误"
			return info, nil
		}

		nowDate := time.Now().Format("20060102")
		currentDate, err := strconv.Atoi(nowDate)
		if err != nil {
			info.ErrorMessage = "系统时间格式错误"
			return info, nil
		}

		// 计算剩余天数
		licTime, err := time.Parse("20060102", conf.LimitedTime)
		if err == nil {
			now := time.Now()
			diff := licTime.Sub(now)
			info.DaysRemaining = int(diff.Hours() / 24)
		}

		if licDate < currentDate {
			info.Status = "expired"
			info.ErrorMessage = fmt.Sprintf("许可证已过期！到期日期: %s, 当前日期: %s", conf.LimitedTime, nowDate)
			return info, nil
		}
	} else {
		// 如果没有设置到期时间，认为是永久许可证
		info.DaysRemaining = -1 // -1 表示永久
	}

	// 如果到这里，说明许可证有效
	info.Status = "valid"
	info.IsValid = true
	info.ErrorMessage = ""

	return info, nil
}

// GetLicenseInfoFormatted 获取格式化的许可证信息字符串，用于直接显示
func GetLicenseInfoFormatted(licenseFile, key string) (string, error) {
	info, err := GetLicenseInfo(licenseFile, key)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString("=== 许可证信息 ===\n")
	result.WriteString(fmt.Sprintf("应用名称: %s\n", info.AppName))
	result.WriteString(fmt.Sprintf("发布公司: %s\n", info.AppCompany))
	result.WriteString(fmt.Sprintf("授权名称: %s\n", info.AuthorizedName))

	if info.LicenseID != "" {
		result.WriteString(fmt.Sprintf("许可证ID: %s\n", info.LicenseID))
	}

	if info.LicenseQuantity > 0 {
		result.WriteString(fmt.Sprintf("许可证数量: %d\n", info.LicenseQuantity))
	}

	result.WriteString(fmt.Sprintf("目标设备UUID: %s\n", info.ObjUUID))
	result.WriteString(fmt.Sprintf("应用UUID: %s\n", info.AppUUID))

	if info.LimitedTime != "" {
		result.WriteString(fmt.Sprintf("到期时间: %s\n", info.LimitedTime))
		if info.DaysRemaining >= 0 {
			result.WriteString(fmt.Sprintf("剩余天数: %d天\n", info.DaysRemaining))
		} else {
			result.WriteString("有效期: 永久\n")
		}
	} else {
		result.WriteString("有效期: 永久\n")
	}

	// 状态信息
	result.WriteString(fmt.Sprintf("许可证状态: "))
	switch info.Status {
	case "valid":
		result.WriteString("✅ 有效\n")
	case "expired":
		result.WriteString("❌ 已过期\n")
	case "invalid":
		result.WriteString("❌ 无效\n")
	}

	if info.ErrorMessage != "" {
		result.WriteString(fmt.Sprintf("错误信息: %s\n", info.ErrorMessage))
	}

	return result.String(), nil
}
