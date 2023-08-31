package LicenseManager

import (
	"fmt"
	"testing"
)

func TestValidAppLic(t *testing.T) {

	licFilePath := "app.lic"
	lic, err := ValidAppLic(licFilePath, "1234567890123456")
	if err != nil {
		t.Log(err.Error())
	}
	if lic {
		fmt.Println("License is valid!")
	} else {
		fmt.Println("License is invalid!")
	}
	t.Log("TestValidAppLic OK!")
}

func TestEncryptLic(t *testing.T) {
	lic, err := ValidAppLic("app.lic", "1234567890123456")
	if err != nil {
		t.Log(err.Error())
	}
	if lic {
		fmt.Println("License is valid!")
	} else {
		fmt.Println("License is invalid!")
	}
	t.Log(lic)

}
