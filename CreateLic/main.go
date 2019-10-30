package main

import (
	"flag"
	"fmt"
	"github.com/xinjiayu/LicenseManager"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"
)

const version = "1.0.0"

func main() {

	var lic_app_name string
	flag.StringVar(&lic_app_name, "name", "appname", "name")
	var lic_date string
	flag.StringVar(&lic_date, "date", "20280130", "date")

	flag.Parse()

	// fmt.Println("lic_app_name:", lic_app_name)
	// fmt.Println("lic_date:", lic_date)
	// os.Exit(0)
	if utf8.RuneCountInString(lic_date) != 8 {
		fmt.Fprintf(os.Stderr, "lic date invalid: %s", lic_date)
	}

	lic_text := LicenseManager.EncryptLic(lic_app_name, lic_date)

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
		// return 2
	}
	dstFile.WriteString(lic_text)
	dstFile.Close()

	fmt.Println("Your lic is ready:", lic_file_path)
	// return 0

}
