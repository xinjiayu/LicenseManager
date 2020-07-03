package main

import (
	"github.com/xinjiayu/LicenseManager"
	"log"
)

func main() {
	key := "123456781234567812345678"

	//LicenseManager.EncryptLic("/Users/microrain/goitem/LicenseManager/testaa/aaa.json",key)

	LicenseManager.ValidAppLic("/Users/microrain/goitem/LicenseManager/app.lic", key)

	log.Println("继续运行。。。")

}
