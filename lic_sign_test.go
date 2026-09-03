package LicenseManager

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xinjiayu/LicenseManager/utils"
)

// signTestKeys 为签名相关测试生成一对密钥
func signTestKeys(t *testing.T) (privatePEM, publicPEM string) {
	t.Helper()
	privatePEM, publicPEM, err := GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}
	return privatePEM, publicPEM
}

func TestGenerateSigningKeyPair(t *testing.T) {
	privatePEM, publicPEM, err := GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}
	if !strings.Contains(privatePEM, "BEGIN PRIVATE KEY") {
		t.Fatalf("私钥PEM格式异常:\n%s", privatePEM)
	}
	if !strings.Contains(publicPEM, "BEGIN PUBLIC KEY") {
		t.Fatalf("公钥PEM格式异常:\n%s", publicPEM)
	}

	// 每次生成的密钥对应不同
	privatePEM2, _, err := GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("生成第二对密钥失败: %v", err)
	}
	if privatePEM == privatePEM2 {
		t.Fatal("两次生成的私钥不应相同")
	}
}

// TestEncryptLicSignedAndValid v2格式：签名、验签、设备与时间校验的完整流程
func TestEncryptLicSignedAndValid(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")

	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if err != nil || !ok {
		t.Fatalf("验证签名许可证失败: ok=%v err=%v", ok, err)
	}
}

// TestSignedLicenseTamperedPayload 篡改负载内容后验签必须失败
func TestSignedLicenseTamperedPayload(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	// 读取信封，将payload中的应用名改掉再写回，签名保持不变
	content, err := os.ReadFile(licFile)
	if err != nil {
		t.Fatalf("读取许可证失败: %v", err)
	}
	var envelope licenseEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("解析许可证信封失败: %v", err)
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		t.Fatalf("解码payload失败: %v", err)
	}
	payload = []byte(strings.Replace(string(payload), "TestApp", "HackedApp", 1))
	envelope.Payload = base64.StdEncoding.EncodeToString(payload)
	tampered, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("重新序列化失败: %v", err)
	}
	if err := os.WriteFile(licFile, tampered, 0644); err != nil {
		t.Fatalf("写入篡改许可证失败: %v", err)
	}

	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if ok || !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("篡改后的许可证验签应失败: ok=%v err=%v", ok, err)
	}
}

// TestSignedLicenseWrongKey 用错误的公钥验签必须失败
func TestSignedLicenseWrongKey(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, _ := signTestKeys(t)
	_, otherPublicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	ok, err := ValidAppLicSigned(licFile, otherPublicPEM)
	if ok || !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("错误公钥验签应失败: ok=%v err=%v", ok, err)
	}
}

func TestSignedLicenseExpired(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, pastDate())
	licFile := filepath.Join(dir, "expired.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if ok || !errors.Is(err, ErrLicenseExpired) {
		t.Fatalf("过期签名许可证应报过期错误: ok=%v err=%v", ok, err)
	}
}

func TestSignedLicenseWrongDevice(t *testing.T) {
	machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, "00000000-0000-0000-0000-000000000000", futureDate())
	licFile := filepath.Join(dir, "other-device.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if ok || !errors.Is(err, ErrDeviceMismatch) {
		t.Fatalf("其他设备的签名许可证应报设备不匹配: ok=%v err=%v", ok, err)
	}
}

func TestGetLicenseInfoSigned(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	info, err := GetLicenseInfoSigned(licFile, publicPEM)
	if err != nil {
		t.Fatalf("获取签名许可证信息失败: %v", err)
	}
	if !info.IsValid || info.Status != "valid" {
		t.Fatalf("期望有效许可证: status=%s err=%s", info.Status, info.ErrorMessage)
	}
	if info.AppName != "TestApp" {
		t.Fatalf("应用名称不匹配: %s", info.AppName)
	}
	if info.LicenseID != "LIC-TEST-001" {
		t.Fatalf("扩展字段LicenseID未解析: %q", info.LicenseID)
	}
	if info.DaysRemaining <= 0 {
		t.Fatalf("剩余天数应大于0: %d", info.DaysRemaining)
	}
}

func TestGetLicenseInfoSignedFormatted(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	formatted, err := GetLicenseInfoSignedFormatted(licFile, publicPEM)
	if err != nil {
		t.Fatalf("获取格式化信息失败: %v", err)
	}
	if !strings.Contains(formatted, "TestApp") {
		t.Fatalf("格式化输出应包含应用名称:\n%s", formatted)
	}
}

// TestEncryptLicSignedBadConfig 配置缺失必要字段时应报错
func TestEncryptLicSignedBadConfig(t *testing.T) {
	privatePEM, _ := signTestKeys(t)
	dir := t.TempDir()
	config := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(config, []byte(`{"AppName":""}`), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	err := EncryptLicSigned(config, filepath.Join(dir, "out.lic"), privatePEM)
	if err == nil || !strings.Contains(err.Error(), "AppName") {
		t.Fatalf("缺失AppName应报错，实际: %v", err)
	}
}

// TestEncryptLicSignedUniversal 通用许可证：可空ObjUUID签发、禁永久、设备校验按预期拒绝
func TestEncryptLicSignedUniversal(t *testing.T) {
	machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)
	dir := t.TempDir()

	// 空ObjUUID + 有到期时间 → 签发成功
	config := writeTestConfig(t, "", futureDate())
	licFile := filepath.Join(dir, "universal.lic")
	if err := EncryptLicSignedUniversal(config, licFile, privatePEM); err != nil {
		t.Fatalf("通用许可证签发失败: %v", err)
	}

	// 仅验签解析：字段正确、不设备绑定判定
	conf, err := ParseSignedLicenseFile(licFile, publicPEM)
	if err != nil {
		t.Fatalf("通用许可证验签失败: %v", err)
	}
	if conf.AppName != "TestApp" || conf.ObjUUID != "" || conf.LimitedTime == "" {
		t.Fatalf("通用许可证字段异常: %+v", conf)
	}

	// 全量验证入口应因设备绑定被拒绝（预期行为，通用授权走 ParseSignedLicense* 自行分级）
	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if ok || !errors.Is(err, ErrDeviceMismatch) {
		t.Fatalf("通用许可证走设备校验应报ErrDeviceMismatch: ok=%v err=%v", ok, err)
	}

	// 缺少LimitedTime → 禁止签发永久通用许可证（哨兵错误可编程识别）
	permConfig := writeTestConfig(t, "", "")
	err = EncryptLicSignedUniversal(permConfig, filepath.Join(dir, "bad.lic"), privatePEM)
	if !errors.Is(err, ErrUniversalRequiresExpiry) {
		t.Fatalf("永久通用许可证应报ErrUniversalRequiresExpiry，实际: %v", err)
	}

	// universal模式下ObjUUID非空 → 拒绝（防止"以为绑定了设备其实没绑"的误判）
	boundConfig := writeTestConfig(t, "some-device-uuid", futureDate())
	err = EncryptLicSignedUniversal(boundConfig, filepath.Join(dir, "bound.lic"), privatePEM)
	if err == nil || !strings.Contains(err.Error(), "ObjUUID") {
		t.Fatalf("通用许可证ObjUUID非空应被拒绝，实际: %v", err)
	}
}

// TestParseSignedLicenseContent 仅验签解析入口：合法/篡改/错误公钥/v1内容
func TestParseSignedLicenseContent(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}
	content, err := os.ReadFile(licFile)
	if err != nil {
		t.Fatalf("读取许可证失败: %v", err)
	}

	// 合法签名 → 字段正确
	conf, err := ParseSignedLicenseContent(content, publicPEM)
	if err != nil {
		t.Fatalf("验签解析失败: %v", err)
	}
	if conf.AppName != "TestApp" || conf.LicenseID != "LIC-TEST-001" {
		t.Fatalf("解析字段异常: %+v", conf)
	}

	// 篡改payload → 验签失败
	var envelope licenseEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("解析信封失败: %v", err)
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		t.Fatalf("解码payload失败: %v", err)
	}
	envelope.Payload = base64.StdEncoding.EncodeToString(
		[]byte(strings.Replace(string(payload), "TestApp", "HackedApp", 1)))
	tampered, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("序列化篡改内容失败: %v", err)
	}
	if _, err := ParseSignedLicenseContent(tampered, publicPEM); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("篡改后应报ErrInvalidSignature，实际: %v", err)
	}

	// 错误公钥 → 验签失败
	_, otherPublicPEM := signTestKeys(t)
	if _, err := ParseSignedLicenseContent(content, otherPublicPEM); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("错误公钥应报ErrInvalidSignature，实际: %v", err)
	}

	// v1 AES内容 → 格式错误
	v1Content := []byte(utils.AesEncrypt(`{"AppName":"LegacyApp"}`, testAESKey))
	if _, err := ParseSignedLicenseContent(v1Content, publicPEM); err == nil || !strings.Contains(err.Error(), "格式错误") {
		t.Fatalf("v1 AES内容应报格式错误，实际: %v", err)
	}
}
