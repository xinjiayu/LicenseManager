package main

import (
	"fmt"
	"github.com/xinjiayu/LicenseManager"
)

func main() {

	fmt.Println("test license OK")
	//======================进行license控制作===========================================
	LicenseManager.ValidAppLic("appname01", "/Users/microrain/goitem/LicenseManager/test/app.lic")
}
