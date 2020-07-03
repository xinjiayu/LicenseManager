package LicenseManager

import (
	"encoding/json"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"github.com/xinjiayu/LicenseManager/utils"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type AppLicenseInfo struct {
	AppName        string //应用名称
	AppCompany     string //应用发布的公司
	AppUUID        string //此次发布应用的UUID
	ObjUUID        string //目标设备的UUID
	AuthorizedName string //授权名称
	LimitedTime    string //到期日期
}

//EncryptLic 跟据应用信息的配置文件生成license授权文件
func EncryptLic(appInfoFile, key string) {
	//从文件中读取配置
	file, err := os.OpenFile(appInfoFile, os.O_RDONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	contentByte, err2 := ioutil.ReadAll(file)
	if err2 != nil {
		panic(err)
	}
	conf := AppLicenseInfo{}
	if err := json.Unmarshal(contentByte, &conf); err == nil {
		tmpText := string(contentByte)
		//进行加密
		tmpText = utils.AesEncrypt(tmpText, key)

		//生成license授权文件
		currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		lic_file_path := currentDir + string(os.PathSeparator) + "app.lic"
		log.Println(lic_file_path)
		lic_file_path = "app.lic"
		dstFile, err := os.Create(lic_file_path)
		if err != nil {
			log.Fatal(err)
		}

		dstFile.WriteString(tmpText)
		dstFile.Close()
	} else {
		fmt.Println(err)
	}

}

func ValidAppLic(appInfoFile, key string) {
	file, err := os.OpenFile(appInfoFile, os.O_RDONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	contentByte, err2 := ioutil.ReadAll(file)
	if err2 != nil {
		panic(err)
	}

	tmpText := string(contentByte)

	//进行解密
	tmpText = utils.AesDecrypt(tmpText, key)
	conf := AppLicenseInfo{}
	if err := json.Unmarshal([]byte(tmpText), &conf); err == nil {

		//获取本机的ID
		id, err := machineid.ID()
		if err != nil {
			log.Fatal(err)
		}

		if conf.ObjUUID != id {
			fmt.Println("001", "授权失败")
			os.Exit(0)

		}

		limitedTime := conf.LimitedTime

		if limitedTime != "" {
			licDate, _ := strconv.Atoi(limitedTime)
			nowDate := time.Now().Format("20060102")
			currentDate, _ := strconv.Atoi(nowDate)
			if licDate < currentDate {
				log.Print("授权结束日期:", licDate)
				log.Fatal("[警告] 授权文件已过期!")
				os.Exit(0)

			}
		}

	} else {
		log.Fatal(err)
	}
}
