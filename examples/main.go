package main

import (
	"fmt"
	"github.com/xinjiayu/LicenseManager"
)

func main() {

	//======================进行license控制作===========================================
	lic, err := LicenseManager.ValidAppLic("app.lic", "558FEC81051A2020")
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
