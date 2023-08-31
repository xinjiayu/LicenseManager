package LicenseManager

import (
	"fmt"
	"testing"
)

func TestValidAppLic(t *testing.T) {

	licFilePath := "app.lic"
	lic, err := ValidAppLic(licFilePath, "1234567890123456")
	if lic {
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		// 授权成功
		t.Log("授权有效")
		return
	}
	t.Log("授权无效，请联系管理员！")
}

func TestEncryptLic(t *testing.T) {

	licFilePath := "aaa.json"
	EncryptLic(licFilePath, "1234567890123456")
	t.Log("success")
}
