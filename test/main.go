package main

import (
	"fmt"
	"github.com/xinjiayu/LicenseManager"
)

func main() {

	//======================进行license控制作===========================================
	LicenseManager.ValidAppLic("app.lic", "558FEC81051A2020")

	fmt.Println("test license OK")

}
