package LicenseManager

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager/utils"
)

const testAESKey = "1234567890123456"

// writeTestConfig 生成一份测试用的授权配置JSON文件，返回文件路径
func writeTestConfig(t *testing.T, objUUID, limitedTime string) string {
	t.Helper()
	conf := map[string]interface{}{
		"AppName":        "TestApp",
		"AppCompany":     "Test Company",
		"AppUUID":        "app-uuid-001",
		"ObjUUID":        objUUID,
		"AuthorizedName": "测试客户",
		"LimitedTime":    limitedTime,
		"LicenseID":      "LIC-TEST-001",
	}
	data, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("生成测试配置失败: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	return path
}

// machineUUIDForTest 获取本机UUID并将防回拨状态文件指向临时目录（避免测试污染真实环境），
// 获取失败时跳过依赖设备匹配的测试
func machineUUIDForTest(t *testing.T) string {
	t.Helper()
	useTempClockState(t)
	id, err := machineid.ID()
	if err != nil {
		t.Skipf("无法获取本机UUID，跳过设备匹配相关测试: %v", err)
	}
	return id
}

func futureDate() string {
	return time.Now().AddDate(1, 0, 0).Format("20060102")
}

func pastDate() string {
	return time.Now().AddDate(-1, 0, 0).Format("20060102")
}

// TestEncryptLicToFileAndValid v1格式：生成并验证许可证的完整流程
func TestEncryptLicToFileAndValid(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "app.lic")

	if err := EncryptLicToFile(config, licFile, testAESKey); err != nil {
		t.Fatalf("生成许可证失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if err != nil || !ok {
		t.Fatalf("验证许可证失败: ok=%v err=%v", ok, err)
	}
}

func TestValidAppLicExpired(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, pastDate())
	licFile := filepath.Join(dir, "expired.lic")

	if err := EncryptLicToFile(config, licFile, testAESKey); err != nil {
		t.Fatalf("生成许可证失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if ok {
		t.Fatal("过期许可证不应验证通过")
	}
	if !errors.Is(err, ErrLicenseExpired) {
		t.Fatalf("期望过期错误，实际: %v", err)
	}
}

func TestValidAppLicWrongDevice(t *testing.T) {
	machineUUIDForTest(t) // 确保能获取本机UUID，否则跳过
	dir := t.TempDir()
	config := writeTestConfig(t, "00000000-0000-0000-0000-000000000000", futureDate())
	licFile := filepath.Join(dir, "other-device.lic")

	if err := EncryptLicToFile(config, licFile, testAESKey); err != nil {
		t.Fatalf("生成许可证失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if ok {
		t.Fatal("其他设备的许可证不应验证通过")
	}
	if !errors.Is(err, ErrDeviceMismatch) {
		t.Fatalf("期望设备不匹配错误，实际: %v", err)
	}
}

// TestValidAppLicCorruptedFile 损坏的授权文件应返回错误而不是panic
func TestValidAppLicCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	licFile := filepath.Join(dir, "broken.lic")
	if err := os.WriteFile(licFile, []byte("not-a-valid-license-content"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if ok || err == nil {
		t.Fatalf("损坏文件应返回错误: ok=%v err=%v", ok, err)
	}
}

// TestValidAppLicInvalidDate 到期时间为非法日期（如不存在的13月）时，
// 签发端与验证端都应拒绝，而不是被当作有效或永久许可证
func TestValidAppLicInvalidDate(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, "20261301")

	// 签发端：格式校验前移，直接拒绝
	err := EncryptLicToFile(config, filepath.Join(dir, "bad-date.lic"), testAESKey)
	if err == nil || !strings.Contains(err.Error(), "到期时间格式错误") {
		t.Fatalf("非法到期日期在签发时就应被拒绝，实际: %v", err)
	}

	// 验证端：手工构造含非法日期的v1许可证（绕过签发校验），验证同样拒绝
	payload := fmt.Sprintf(`{"AppName":"TestApp","ObjUUID":%q,"LimitedTime":"20261301"}`, uuid)
	licFile := filepath.Join(dir, "handmade.lic")
	if err := os.WriteFile(licFile, []byte(utils.AesEncrypt(payload, testAESKey)), 0644); err != nil {
		t.Fatalf("写入测试许可证失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if ok || err == nil || !strings.Contains(err.Error(), "到期时间格式错误") {
		t.Fatalf("非法到期日期应报格式错误: ok=%v err=%v", ok, err)
	}
}

func TestGetLicenseInfo(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "app.lic")

	if err := EncryptLicToFile(config, licFile, testAESKey); err != nil {
		t.Fatalf("生成许可证失败: %v", err)
	}

	info, err := GetLicenseInfo(licFile, testAESKey)
	if err != nil {
		t.Fatalf("获取许可证信息失败: %v", err)
	}
	if !info.IsValid || info.Status != "valid" {
		t.Fatalf("期望有效许可证，实际: status=%s err=%s", info.Status, info.ErrorMessage)
	}
	if info.AppName != "TestApp" {
		t.Fatalf("应用名称不匹配: %s", info.AppName)
	}
	if info.LicenseID != "LIC-TEST-001" {
		t.Fatalf("扩展字段LicenseID未解析: %q", info.LicenseID)
	}
	if info.DaysRemaining <= 0 {
		t.Fatalf("剩余天数应大于0，实际: %d", info.DaysRemaining)
	}
}

func TestGetLicenseInfoFormatted(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "app.lic")

	if err := EncryptLicToFile(config, licFile, testAESKey); err != nil {
		t.Fatalf("生成许可证失败: %v", err)
	}

	formatted, err := GetLicenseInfoFormatted(licFile, testAESKey)
	if err != nil {
		t.Fatalf("获取格式化信息失败: %v", err)
	}
	if !strings.Contains(formatted, "TestApp") {
		t.Fatalf("格式化输出应包含应用名称:\n%s", formatted)
	}
}

// TestLicenseV1CompatLegacyFixedIV 验证v1旧格式（无IV前缀、密钥作IV）的许可证文件仍可通过ValidAppLic验证
func TestLicenseV1CompatLegacyFixedIV(t *testing.T) {
	uuid := machineUUIDForTest(t)
	dir := t.TempDir()

	// 构造旧格式许可证：明文直接用密钥前16字节作为IV加密（无随机IV前缀）
	conf := map[string]interface{}{
		"AppName":     "LegacyApp",
		"ObjUUID":     uuid,
		"LimitedTime": futureDate(),
	}
	plaintext, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("生成配置失败: %v", err)
	}
	k := []byte(testAESKey)
	padNum := aes.BlockSize - len(plaintext)%aes.BlockSize
	data := append(plaintext, bytes.Repeat([]byte{byte(padNum)}, padNum)...)
	block, err := aes.NewCipher(k)
	if err != nil {
		t.Fatalf("创建AES块失败: %v", err)
	}
	out := make([]byte, len(data))
	cipher.NewCBCEncrypter(block, k[:aes.BlockSize]).CryptBlocks(out, data)

	licFile := filepath.Join(dir, "legacy.lic")
	if err := os.WriteFile(licFile, []byte(base64.StdEncoding.EncodeToString(out)), 0644); err != nil {
		t.Fatalf("写入旧格式许可证失败: %v", err)
	}

	ok, err := ValidAppLic(licFile, testAESKey)
	if err != nil || !ok {
		t.Fatalf("旧格式许可证应验证通过: ok=%v err=%v", ok, err)
	}
}
