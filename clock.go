package LicenseManager

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// 离线许可证的到期校验依赖系统时间，回拨时钟即可绕过。这里在本地记录
// 每个许可证"最后一次通过验证的时间"（只增不减），发现系统时间显著早于
// 该记录时拒绝验证。状态文件本身可被删除，因此该机制只用于抬高攻击
// 门槛，不能完全杜绝回拨；高价值场景建议配合在线激活。

const (
	// clockRollbackTolerance 允许的时钟误差，回拨幅度小于该值不视为攻击
	clockRollbackTolerance = 48 * time.Hour

	clockStateVersion = 1
)

// ErrClockRollback 检测到系统时间相对上次验证记录明显回拨
var ErrClockRollback = errors.New("检测到系统时间回拨，请校正系统时间后重试")

// clockStateFilePath 防回拨状态文件路径，空字符串表示禁用该机制；测试中可替换
var clockStateFilePath = defaultClockStateFilePath()

func defaultClockStateFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".licensemanager", "clock_state.json")
}

// clockStateFile 状态文件结构，按许可证指纹记录最后验证时间
type clockStateFile struct {
	Version  int               `json:"version"`
	LastSeen map[string]string `json:"last_seen"`
}

// licenseFingerprint 以授权信息的关键字段生成指纹，作为状态记录的键，
// 使同一份授权在任何路径下共享同一条记录
func licenseFingerprint(conf *AppLicenseInfo) string {
	sum := sha256.Sum256([]byte(conf.AppName + "|" + conf.AppCompany + "|" +
		conf.AppUUID + "|" + conf.ObjUUID + "|" + conf.AuthorizedName + "|" + conf.LimitedTime))
	return hex.EncodeToString(sum[:])
}

// checkClockAndRecord 校验系统时间未相对上次验证记录回拨，校验通过后更新记录（只增不减）
func checkClockAndRecord(conf *AppLicenseInfo) error {
	// 永久许可证不受到期时间约束，回拨无意义，不记录
	if conf.LimitedTime == "" {
		return nil
	}

	state := loadClockState()
	fingerprint := licenseFingerprint(conf)
	now := time.Now().UTC()

	if lastStr, ok := state.LastSeen[fingerprint]; ok {
		if last, err := time.Parse(time.RFC3339, lastStr); err == nil {
			if now.Before(last.Add(-clockRollbackTolerance)) {
				return fmt.Errorf("%w: 上次验证时间 %s，当前系统时间 %s",
					ErrClockRollback,
					last.Local().Format("2006-01-02 15:04:05"),
					now.Local().Format("2006-01-02 15:04:05"))
			}
			// 只增不减：不把记录时间往回调
			if !now.After(last) {
				return nil
			}
		}
	}

	state.Version = clockStateVersion
	if state.LastSeen == nil {
		state.LastSeen = map[string]string{}
	}
	state.LastSeen[fingerprint] = now.Format(time.RFC3339)
	saveClockState(state)
	return nil
}

// loadClockState 读取状态文件，禁用、缺失或损坏时返回空状态，不阻塞验证
func loadClockState() clockStateFile {
	state := clockStateFile{Version: clockStateVersion, LastSeen: map[string]string{}}
	if clockStateFilePath == "" {
		return state
	}
	data, err := os.ReadFile(clockStateFilePath)
	if err != nil {
		return state
	}
	var parsed clockStateFile
	if json.Unmarshal(data, &parsed) != nil || parsed.LastSeen == nil {
		return state
	}
	return parsed
}

// saveClockState 尽力而为地保存状态文件，失败不影响验证结果
func saveClockState(state clockStateFile) {
	if clockStateFilePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(clockStateFilePath), 0700); err != nil {
		return
	}
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = os.WriteFile(clockStateFilePath, data, 0600)
}
