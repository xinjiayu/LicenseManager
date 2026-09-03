package LicenseManager

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// v2 许可证采用非对称签名：签发方持有 Ed25519 私钥，客户端只需内嵌公钥即可验证，
// 客户端拿不到私钥，因此无法伪造或篡改许可证内容（v1 的 AES 对称加密则相反，
// 验证密钥就是加密密钥，逆向客户端即可自行生成许可证）。

const (
	// LicenseAlgEd25519 v2 许可证使用的签名算法
	LicenseAlgEd25519 = "Ed25519"

	// licenseVersionV2 带签名的许可证格式版本
	licenseVersionV2 = 2
)

var (
	// ErrInvalidSignature 许可证签名验证失败（内容被篡改或公钥不匹配）
	ErrInvalidSignature = errors.New("许可证签名验证失败，文件可能被篡改")
	// ErrUnsupportedLicense 许可证算法或版本不受支持
	ErrUnsupportedLicense = errors.New("许可证算法或版本不受支持")
	// ErrUniversalRequiresExpiry 通用许可证缺少到期时间（禁止签发永久通用许可证）
	ErrUniversalRequiresExpiry = errors.New("通用许可证必须设置到期时间")
)

// licenseEnvelope v2 许可证文件结构。payload 为 base64 编码的授权信息 JSON 原始字节，
// signature 为 Ed25519 对 payload 解码后字节的签名，同样 base64 编码。
// 注意：IssuedAt 不在签名覆盖范围内，仅作信息展示，可被篡改，禁止用于任何安全判定。
type licenseEnvelope struct {
	Version   int    `json:"version"`
	Alg       string `json:"alg"`
	IssuedAt  string `json:"issued_at"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// GenerateSigningKeyPair 生成一对 Ed25519 密钥，返回 PEM 编码（PKCS#8 私钥、PKIX 公钥）。
// 私钥必须由签发方妥善保管，公钥内嵌到客户端用于验证许可证。
func GenerateSigningKeyPair() (privatePEM, publicPEM string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("生成Ed25519密钥对失败: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("编码私钥失败: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("编码公钥失败: %w", err)
	}

	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))
	publicPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privatePEM, publicPEM, nil
}

// parseSigningPrivateKey 从 PEM 文本解析 Ed25519 私钥
func parseSigningPrivateKey(privatePEM string) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return nil, errors.New("私钥PEM格式错误")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析Ed25519私钥失败: %w", err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("密钥类型错误，需要Ed25519私钥")
	}
	return priv, nil
}

// parseSigningPublicKey 从 PEM 文本解析 Ed25519 公钥
func parseSigningPublicKey(publicPEM string) (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicPEM))
	if block == nil {
		return nil, errors.New("公钥PEM格式错误")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析Ed25519公钥失败: %w", err)
	}
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("密钥类型错误，需要Ed25519公钥")
	}
	return pub, nil
}

// EncryptLicSigned 读取 JSON 授权配置文件，用 Ed25519 私钥对其签名并生成 v2 许可证文件。
// 签名覆盖配置文件的原始字节，验证端以文件内嵌的 payload 为准，不依赖重新序列化。
func EncryptLicSigned(appInfoFile, output, privateKeyPEM string) error {
	return signLicenseToFile(appInfoFile, output, privateKeyPEM, false)
}

// EncryptLicSignedUniversal 签发通用许可证（ObjUUID 留空 = 不绑定设备，任何机器可验签）。
// 通用许可证不绑定设备、一旦外泄即全网可用，因此禁止签发永久通用许可证：
// LimitedTime 必填，到期后必须重新签发。调用方可在其之上叠加更严格的产品规则
// （如有效期上限），本库只保证「通用必有过期时间」这一安全底线。
// 通用许可证不做设备绑定判定，验证侧请配合 ParseSignedLicenseFile/Content
// 实现自定义校验；通过 ValidAppLicSigned 验证会因 ErrDeviceMismatch 被拒绝。
func EncryptLicSignedUniversal(appInfoFile, output, privateKeyPEM string) error {
	return signLicenseToFile(appInfoFile, output, privateKeyPEM, true)
}

// signLicenseToFile 签发 v2 许可证的共用实现。
// universal 为 true 时允许 ObjUUID 留空（通用许可证），但要求 LimitedTime 必填。
func signLicenseToFile(appInfoFile, output, privateKeyPEM string, universal bool) error {
	content, err := os.ReadFile(appInfoFile)
	if err != nil {
		return fmt.Errorf("打开配置文件失败: %w", err)
	}

	var conf AppLicenseInfo
	if err := json.Unmarshal(content, &conf); err != nil {
		return fmt.Errorf("解析配置文件失败，请检查JSON格式: %w", err)
	}
	if conf.AppName == "" {
		return errors.New("配置文件中AppName字段不能为空")
	}
	if universal && conf.ObjUUID != "" {
		return errors.New("通用许可证的ObjUUID必须留空（如需绑定设备请使用EncryptLicSigned）")
	}
	if !universal && conf.ObjUUID == "" {
		return errors.New("配置文件中ObjUUID字段不能为空")
	}
	if universal && conf.LimitedTime == "" {
		return fmt.Errorf("%w（LimitedTime），禁止签发永久通用许可证", ErrUniversalRequiresExpiry)
	}
	if conf.LimitedTime != "" {
		if _, err := time.Parse("20060102", conf.LimitedTime); err != nil {
			return fmt.Errorf("到期时间格式错误，应为YYYYMMDD: %w", err)
		}
	}

	priv, err := parseSigningPrivateKey(privateKeyPEM)
	if err != nil {
		return err
	}

	payload := content
	signature := ed25519.Sign(priv, payload)

	envelope := licenseEnvelope{
		Version:   licenseVersionV2,
		Alg:       LicenseAlgEd25519,
		IssuedAt:  time.Now().UTC().Format(time.RFC3339),
		Payload:   base64.StdEncoding.EncodeToString(payload),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("生成许可证内容失败: %w", err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("写入授权文件失败: %w", err)
	}
	return nil
}

// ParseSignedLicenseContent 验签并解析 v2 许可证内容（[]byte），
// 不做设备/到期/回拨判定——供调用方实现自定义分级校验
// （如通用授权跳过设备绑定、内嵌字符串授权等场景）。
func ParseSignedLicenseContent(content []byte, publicKeyPEM string) (*AppLicenseInfo, error) {
	if len(content) == 0 {
		return nil, errors.New("授权文件为空")
	}

	var envelope licenseEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, fmt.Errorf("授权文件格式错误（非v2签名格式）: %w", err)
	}
	if envelope.Alg != LicenseAlgEd25519 || envelope.Version != licenseVersionV2 {
		return nil, fmt.Errorf("%w: alg=%s version=%d", ErrUnsupportedLicense, envelope.Alg, envelope.Version)
	}

	pub, err := parseSigningPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, fmt.Errorf("许可证负载解码失败: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return nil, fmt.Errorf("许可证签名解码失败: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("%w: 签名长度为%d，应为%d", ErrInvalidSignature, len(signature), ed25519.SignatureSize)
	}
	if !ed25519.Verify(pub, payload, signature) {
		return nil, ErrInvalidSignature
	}

	var conf AppLicenseInfo
	if err := json.Unmarshal(payload, &conf); err != nil {
		return nil, errors.New("授权文件格式错误或签名内容异常")
	}
	return &conf, nil
}

// ParseSignedLicenseFile 验签并解析 v2 许可证文件，
// 不做设备/到期/回拨判定，用途同 ParseSignedLicenseContent。
func ParseSignedLicenseFile(licFile, publicKeyPEM string) (*AppLicenseInfo, error) {
	content, err := os.ReadFile(licFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("授权文件不存在")
		}
		return nil, fmt.Errorf("打开授权文件失败: %w", err)
	}
	return ParseSignedLicenseContent(content, publicKeyPEM)
}

// ValidAppLicSigned 验证 v2 签名许可证：先验签，再校验设备UUID与到期时间。
func ValidAppLicSigned(licFile, publicKeyPEM string) (bool, error) {
	conf, err := ParseSignedLicenseFile(licFile, publicKeyPEM)
	if err != nil {
		return false, err
	}
	if err := validateLicenseInfo(conf); err != nil {
		return false, err
	}
	return true, nil
}

// GetLicenseInfoSigned 获取 v2 签名许可证的详细信息，用于界面显示。
// 与 GetLicenseInfo 一样以结构体承载验证结果，验证过程本身出错时才返回 error。
func GetLicenseInfoSigned(licFile, publicKeyPEM string) (*LicenseDisplayInfo, error) {
	info := &LicenseDisplayInfo{
		Status:        "invalid",
		IsValid:       false,
		DaysRemaining: -1,
	}

	conf, err := ParseSignedLicenseFile(licFile, publicKeyPEM)
	if err != nil {
		info.ErrorMessage = err.Error()
		return info, nil
	}

	fillDisplayInfo(info, conf)

	if err := validateLicenseInfo(conf); err != nil {
		info.Status = licenseStatusForError(err)
		info.ErrorMessage = err.Error()
		return info, nil
	}

	info.Status = "valid"
	info.IsValid = true
	info.ErrorMessage = ""
	return info, nil
}

// GetLicenseInfoSignedFormatted 获取 v2 签名许可证的格式化文本信息，用于直接显示。
func GetLicenseInfoSignedFormatted(licFile, publicKeyPEM string) (string, error) {
	info, err := GetLicenseInfoSigned(licFile, publicKeyPEM)
	if err != nil {
		return "", err
	}
	return formatLicenseInfo(info), nil
}
