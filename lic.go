package LicenseManager

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	"log"
)

var encryptSalt = "Antell Technolony Co.,Ltd"

//EncryptLic 加密授权信息
func EncryptLic(licAppName string, date string) string {
	appNameMd5 := makeMd5(licAppName + encryptSalt)
	dateEncrypt := desEncrypt([]byte(date), []byte(encryptSalt))
	dateEncryptHexStr := bytesToHexString(dateEncrypt)
	md5Check := makeMd5(licAppName + date + appNameMd5 + dateEncryptHexStr + encryptSalt)
	return licAppName + date + appNameMd5 + dateEncryptHexStr + md5Check
}

//DecryptLic 解密授权信息
func DecryptLic(licAppName string, lic string) string {
	if len(lic) <= 88 {
		log.Fatal(fmt.Sprintf("[L000] 授权无效: %s", lic))
	}
	licRunes := []rune(lic)
	licLen := len(licRunes)
	if makeMd5(string(licRunes[0:licLen-32])+encryptSalt) != string(licRunes[licLen-32:licLen]) {
		log.Fatal(fmt.Sprintf("[L001] 授权无效: %s", lic))
	}
	encryptDate := licRunes[licLen-48 : licLen-32]
	decryptDate := string(desDecrypt(hexStringToBytes(string(encryptDate)), []byte(encryptSalt)))
	if makeMd5(licAppName+encryptSalt) != string(licRunes[licLen-80:licLen-48]) {
		log.Fatal(fmt.Sprintf("[L002] 授权无效: %s", lic))
	}
	if string(licRunes[licLen-88:licLen-80]) != decryptDate {
		log.Fatal(fmt.Sprintf("[L003] 授权无效: %s", lic))
	}
	if string(licRunes[0:licLen-88]) != licAppName {
		log.Fatal(fmt.Sprintf("[L004] 授权无效: %s", lic))
	}
	return decryptDate
}

//ValidAppLic 判断授权文件是否有效
func ValidAppLic(licAppName string, licFilePath string) {
	licBytes, err := ioutil.ReadFile(licFilePath)
	if err != nil {
		log.Fatal(fmt.Sprintf("[L005] license文件未发现: %s", licFilePath))
	}
	licDate := DecryptLic(licAppName, string(licBytes))
	currentDate := time.Now().Format("20060102")
	if licDate < currentDate {
		log.Print("授权结束日期:", licDate)
		// log.Fatal(fmt.Sprintf("[L006] lic not valid: %s", string(lic_bytes)))
		log.Fatal("[警告] 授权文件已过期!")

	}
}

func makeMd5(clearText string) string {
	h := md5.New()
	h.Write([]byte(clearText))
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func desEncrypt(origData []byte, key []byte) []byte {
	key = makeRightLenDesKey(key)
	block, err := des.NewCipher(key)
	if err != nil {
		panic(err)
	}
	blockMode := cipher.NewCBCEncrypter(block, key)
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted
}

func desDecrypt(crypted []byte, key []byte) []byte {
	key = makeRightLenDesKey(key)
	block, err := des.NewCipher(key)
	if err != nil {
		panic(err)
	}
	blockMode := cipher.NewCBCDecrypter(block, key)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	return origData
}

func makeRightLenDesKey(keyBytes []byte) []byte {
	var desKeyLen = 8
	keyBytesLen := len(keyBytes)
	if keyBytesLen > desKeyLen {
		keyBytes = keyBytes[0:desKeyLen]
	} else if keyBytesLen != desKeyLen {
		keyBytes = append(keyBytes, bytes.Repeat([]byte("0"), desKeyLen-keyBytesLen)...)
	}
	return keyBytes
}

func bytesToHexString(bytesData []byte) string {
	var buffer bytes.Buffer
	bytesDataLen := len(bytesData)
	for i := 0; i < bytesDataLen; i++ {
		hexStr := fmt.Sprintf("%x", bytesData[i])
		if len(hexStr) == 1 {
			buffer.WriteString("0")
		}
		buffer.WriteString(hexStr)
	}
	return buffer.String()
}

func hexStringToBytes(hexStr string) []byte {
	hexRunes := []rune(hexStr)
	hexRunesLen := len(hexRunes)
	var hexBytes = make([]byte, 0)
	for i := 0; i < hexRunesLen/2; i++ {
		b := byte(hexDigitRuneToNum(hexRunes[2*i])*16 + hexDigitRuneToNum(hexRunes[2*i+1]))
		hexBytes = append(hexBytes, b)
	}
	return hexBytes
}

func hexDigitRuneToNum(hexDigitRune rune) int {
	if hexDigitRune >= rune('a') {
		return int(hexDigitRune-rune('a')) + 10
	}
	return int(hexDigitRune - rune('0'))

}
