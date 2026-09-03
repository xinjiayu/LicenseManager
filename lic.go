package LicenseManager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager/utils"
)

var (
	// ErrLicenseExpired 许可证已超过授权有效期
	ErrLicenseExpired = errors.New("许可证已过期")
	// ErrDeviceMismatch 许可证绑定的设备与当前设备不匹配
	ErrDeviceMismatch = errors.New("许可证不适用于此设备")
)

// 机器码在运行期不会合法变化，而部分平台（如macOS的ioreg子进程）每次获取
// 开销不小，验证又可能高频触发，因此成功获取一次后全程缓存。
// 失败不缓存（获取失败可能是子进程偶发超时等瞬时原因），下次验证会重试。
var (
	machineUUIDMu     sync.Mutex
	machineUUID       string
	machineUUIDCached bool
)

// cachedMachineUUID 返回缓存的本机机器码，未缓存或上次失败时重新获取
func cachedMachineUUID() (string, error) {
	machineUUIDMu.Lock()
	defer machineUUIDMu.Unlock()

	if machineUUIDCached {
		return machineUUID, nil
	}

	id, err := machineid.ID()
	if err != nil {
		return "", err
	}
	machineUUID, machineUUIDCached = id, true
	return id, nil
}

type AppLicenseInfo struct {
	AppName         string //应用名称
	AppCompany      string //应用发布的公司
	AppUUID         string //此次发布应用的UUID
	ObjUUID         string //目标设备的UUID
	AuthorizedName  string //授权名称
	LimitedTime     string //到期日期
	LicenseID       string //许可证ID（可选扩展字段）
	LicenseQuantity int    //许可证数量（可选扩展字段）
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
	Status          string `json:"status"`           // 许可证状态：valid, expired, invalid
	DaysRemaining   int    `json:"days_remaining"`   // 剩余天数
	IsValid         bool   `json:"is_valid"`         // 是否有效
	ErrorMessage    string `json:"error_message"`    // 错误信息（如果有）
}

// validateLicenseInfo 校验授权信息是否匹配当前设备与时间，v1/v2 许可证共用
func validateLicenseInfo(conf *AppLicenseInfo) error {
	// 获取本机的ID（运行期缓存，见 cachedMachineUUID）
	id, err := cachedMachineUUID()
	if err != nil {
		return errors.New("获取本机ID失败")
	}

	// 验证设备UUID
	if conf.ObjUUID != id {
		return fmt.Errorf("%w: 授权对象UUID与本机UUID不一致", ErrDeviceMismatch)
	}

	// 验证到期时间（统一按UTC比较，避免跨时区部署时到期日判断出现偏差）
	if conf.LimitedTime != "" {
		// 用 time.Parse 严格校验日期合法性，拒绝 20261301 这类并不存在的日期
		licTime, err := time.Parse("20060102", conf.LimitedTime)
		if err != nil {
			return errors.New("授权文件中的到期时间格式错误")
		}

		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if licTime.Before(today) {
			return fmt.Errorf("%w！授权结束日期: %s, 当前日期: %s", ErrLicenseExpired, conf.LimitedTime, today.Format("20060102"))
		}
	}

	// 检测系统时间回拨并记录本次验证时间
	return checkClockAndRecord(conf)
}

// licenseStatusForError 根据校验错误映射显示状态
func licenseStatusForError(err error) string {
	if errors.Is(err, ErrLicenseExpired) {
		return "expired"
	}
	return "invalid"
}

// daysRemaining 计算距到期时间的剩余天数（按UTC），永久许可证（未设置到期时间）返回 -1
func daysRemaining(limitedTime string) int {
	if limitedTime == "" {
		return -1
	}
	licTime, err := time.Parse("20060102", limitedTime)
	if err != nil {
		return -1
	}
	return int(licTime.Sub(time.Now().UTC()).Hours() / 24)
}

// fillDisplayInfo 将授权信息填入显示结构
func fillDisplayInfo(info *LicenseDisplayInfo, conf *AppLicenseInfo) {
	info.AppName = conf.AppName
	info.AppCompany = conf.AppCompany
	info.AppUUID = conf.AppUUID
	info.ObjUUID = conf.ObjUUID
	info.AuthorizedName = conf.AuthorizedName
	info.LimitedTime = conf.LimitedTime
	info.LicenseID = conf.LicenseID
	info.LicenseQuantity = conf.LicenseQuantity
	info.DaysRemaining = daysRemaining(conf.LimitedTime)
}

// decryptLicense 解密 v1 许可证内容，将底层工具的panic转换为错误，
// 避免密钥错误或文件损坏导致宿主程序崩溃。
// 先按新格式（随机IV前缀）解密，失败再按旧格式（密钥作IV）解密，
// 以"明文可解析为JSON"作为最终判据：CBC下错误的IV只会破坏第一个块、
// 填充校验可能碰巧通过，只有密钥与格式都正确才能解出合法JSON。
func decryptLicense(cipherText, key string) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			text = ""
			err = fmt.Errorf("许可证解密失败，可能密钥错误或文件损坏: %v", r)
		}
	}()

	if plain, decryptErr := utils.AesDecryptPrefixIV(cipherText, key); decryptErr == nil && json.Valid([]byte(plain)) {
		return plain, nil
	}
	if plain, decryptErr := utils.AesDecryptFixedIV(cipherText, key); decryptErr == nil && json.Valid([]byte(plain)) {
		return plain, nil
	}
	return "", errors.New("许可证解密失败，可能密钥错误或文件损坏")
}

// EncryptLic 根据应用信息的配置文件生成license授权文件（固定输出 app.lic）。
//
// Deprecated: 该函数固定输出 app.lic，且失败时以 log.Fatal 终止进程，
// 不适合在库代码中使用。请改用 EncryptLicToFile（可自定义输出路径并返回 error）。
// 对于新签发的许可证，建议使用 Ed25519 签名格式 EncryptLicSigned。
func EncryptLic(appInfoFile, key string) {
	contentByte, err := os.ReadFile(appInfoFile)
	if err != nil {
		log.Fatalf("打开配置文件失败: %v", err)
	}
	var conf AppLicenseInfo
	if err := json.Unmarshal(contentByte, &conf); err != nil {
		log.Fatalf("解析配置文件失败，请检查JSON格式: %v", err)
	}

	log.Printf("应用名称: %s", conf.AppName)
	log.Printf("目标设备UUID: %s", conf.ObjUUID)
	log.Printf("授权到期时间: %s", conf.LimitedTime)

	if err := EncryptLicToFile(appInfoFile, "app.lic", key); err != nil {
		log.Fatal(err)
	}
	log.Printf("授权文件已生成: %s", "app.lic")
}

// EncryptLicToFile 根据应用信息的配置文件生成license授权文件，输出到指定路径
func EncryptLicToFile(appInfoFile, output, key string) error {
	contentByte, err := os.ReadFile(appInfoFile)
	if err != nil {
		return fmt.Errorf("打开配置文件失败: %w", err)
	}

	// 验证JSON格式与必要字段
	var conf AppLicenseInfo
	if err := json.Unmarshal(contentByte, &conf); err != nil {
		return fmt.Errorf("解析配置文件失败，请检查JSON格式: %w", err)
	}
	if conf.AppName == "" {
		return errors.New("配置文件中AppName字段不能为空")
	}
	if conf.ObjUUID == "" {
		return errors.New("配置文件中ObjUUID字段不能为空")
	}
	// 与验证端validateLicenseInfo一致：到期时间非空时必须是合法YYYYMMDD，
	// 避免签出验证端必然拒绝的许可证
	if conf.LimitedTime != "" {
		if _, err := time.Parse("20060102", conf.LimitedTime); err != nil {
			return fmt.Errorf("到期时间格式错误，应为YYYYMMDD: %w", err)
		}
	}

	// 进行加密
	encryptedText := utils.AesEncrypt(string(contentByte), key)

	if err := os.WriteFile(output, []byte(encryptedText), 0644); err != nil {
		return fmt.Errorf("写入授权文件失败: %w", err)
	}
	return nil
}

// ValidAppLic 验证应用许可证（v1 AES对称加密格式）
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
			log.Printf("关闭文件失败: %v", closeErr)
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

	// 进行解密，失败时返回错误而不是panic
	decryptedText, err := decryptLicense(string(contentByte), key)
	if err != nil {
		return false, err
	}

	// 解析JSON
	var conf AppLicenseInfo
	if err := json.Unmarshal([]byte(decryptedText), &conf); err != nil {
		return false, errors.New("授权文件格式错误或解密失败")
	}

	if err := validateLicenseInfo(&conf); err != nil {
		return false, err
	}
	return true, nil
}

// GetLicenseInfo 获取许可证文件的详细信息，用于界面显示（v1 AES对称加密格式）
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

	// 解密，失败时返回错误而不是panic
	decryptedText, err := decryptLicense(string(contentByte), key)
	if err != nil {
		info.ErrorMessage = "许可证文件解密失败，可能密钥错误或文件损坏"
		return info, nil
	}

	// 解析JSON到原始结构
	var conf AppLicenseInfo
	if err := json.Unmarshal([]byte(decryptedText), &conf); err != nil {
		info.ErrorMessage = "许可证文件格式错误"
		return info, nil
	}

	// 填充基本信息
	fillDisplayInfo(info, &conf)

	if err := validateLicenseInfo(&conf); err != nil {
		info.Status = licenseStatusForError(err)
		info.ErrorMessage = err.Error()
		return info, nil
	}

	// 如果到这里，说明许可证有效
	info.Status = "valid"
	info.IsValid = true
	info.ErrorMessage = ""

	return info, nil
}

// GetLicenseInfoFormatted 获取格式化的许可证信息字符串，用于直接显示（v1 AES对称加密格式）
func GetLicenseInfoFormatted(licenseFile, key string) (string, error) {
	info, err := GetLicenseInfo(licenseFile, key)
	if err != nil {
		return "", err
	}
	return formatLicenseInfo(info), nil
}

// formatLicenseInfo 将许可证信息渲染为多行文本
func formatLicenseInfo(info *LicenseDisplayInfo) string {
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
	result.WriteString("许可证状态: ")
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

	return result.String()
}
