package main

import (
	"flag"
	"fmt"
	"github.com/xinjiayu/LicenseManager"
	"os"
	"unicode/utf8"
)

// 实际中应该用更好的变量名
var (
	h bool
	f string
	k string
)

func init() {
	flag.BoolVar(&h, "h", false, "查看帮助信息")
	flag.StringVar(&f, "f", "", "需要授权的应用信息配置文件，json格式")
	flag.StringVar(&k, "k", "1234567890123456", "授权钥匙")
	// 改变默认的 Usage，flag包中的Usage 其实是一个函数类型。这里是覆盖默认函数实现，具体见后面Usage部分的分析
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if h {
		flag.Usage()
	}

	if f != "" && k != "" {
		if utf8.RuneCountInString(k) != 24 {
			fmt.Fprintf(os.Stderr, "key invalid: %s", k)
		}
		LicenseManager.EncryptLic(f, k)
	}

}

func usage() {
	fmt.Fprintf(os.Stderr, `CreateLic version: 1.0.0
Usage: CreateLic [-hfk] [-f filename] [-k key]

Options:
`)
	flag.PrintDefaults()
}
