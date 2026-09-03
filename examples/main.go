package main

import (
	"fmt"
	"github.com/xinjiayu/LicenseManager"
)

func main() {

	//======================进行license控制作===========================================
	// v1 AES格式的调用示例；新项目建议使用 ValidAppLicSigned（Ed25519签名格式）
	lic, err := LicenseManager.ValidAppLic("app.lic", "your-16-byte-key")
	if lic {
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("License is valid!")
	}

	if err != nil {
		fmt.Println(err.Error())
	}

}
