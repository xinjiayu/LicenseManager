package LicenseManager

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// useTempClockState 将防回拨状态文件指向临时目录，避免测试污染真实环境
func useTempClockState(t *testing.T) {
	t.Helper()
	old := clockStateFilePath
	clockStateFilePath = filepath.Join(t.TempDir(), "clock_state.json")
	t.Cleanup(func() { clockStateFilePath = old })
}

// TestClockRollbackDetection 系统时间相对上次验证记录明显回拨时，验证应被拒绝
func TestClockRollbackDetection(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	// 第一次验证：正常通过并记录时间
	if ok, err := ValidAppLicSigned(licFile, publicPEM); err != nil || !ok {
		t.Fatalf("首次验证应通过: ok=%v err=%v", ok, err)
	}

	// 将状态记录改为10天后，等效于系统时间被回拨10天
	raw, err := os.ReadFile(clockStateFilePath)
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}
	var state clockStateFile
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("解析状态文件失败: %v", err)
	}
	if len(state.LastSeen) != 1 {
		t.Fatalf("状态文件应恰好有1条记录，实际%d条", len(state.LastSeen))
	}
	for fingerprint := range state.LastSeen {
		state.LastSeen[fingerprint] = time.Now().UTC().Add(10 * 24 * time.Hour).Format(time.RFC3339)
	}
	patched, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("序列化状态失败: %v", err)
	}
	if err := os.WriteFile(clockStateFilePath, patched, 0600); err != nil {
		t.Fatalf("写回状态文件失败: %v", err)
	}

	// 再次验证：应报时间回拨
	ok, err := ValidAppLicSigned(licFile, publicPEM)
	if ok || !errors.Is(err, ErrClockRollback) {
		t.Fatalf("回拨后验证应被拒绝: ok=%v err=%v", ok, err)
	}
}

// TestClockPermanentLicenseSkipped 永久许可证不受到期时间约束，不应记录防回拨状态
func TestClockPermanentLicenseSkipped(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, "") // 不设置到期时间 => 永久
	licFile := filepath.Join(dir, "perm.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	if ok, err := ValidAppLicSigned(licFile, publicPEM); err != nil || !ok {
		t.Fatalf("永久许可证应验证通过: ok=%v err=%v", ok, err)
	}

	if _, err := os.Stat(clockStateFilePath); !os.IsNotExist(err) {
		t.Fatalf("永久许可证不应写防回拨状态文件")
	}
}

// TestClockStateCorruptFailOpen 状态文件损坏时不应阻塞验证（fail open）
func TestClockStateCorruptFailOpen(t *testing.T) {
	uuid := machineUUIDForTest(t)
	privatePEM, publicPEM := signTestKeys(t)

	dir := t.TempDir()
	config := writeTestConfig(t, uuid, futureDate())
	licFile := filepath.Join(dir, "signed.lic")
	if err := EncryptLicSigned(config, licFile, privatePEM); err != nil {
		t.Fatalf("生成签名许可证失败: %v", err)
	}

	if err := os.WriteFile(clockStateFilePath, []byte("not-json{{{"), 0600); err != nil {
		t.Fatalf("写入损坏状态失败: %v", err)
	}

	if ok, err := ValidAppLicSigned(licFile, publicPEM); err != nil || !ok {
		t.Fatalf("状态损坏时验证应正常通过: ok=%v err=%v", ok, err)
	}
}
