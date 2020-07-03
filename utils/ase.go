package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"log"
)

//加密
func AesEncrypt(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)

	block, err := aes.NewCipher(k)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	origData = padding(origData, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, k)
	blockMode.CryptBlocks(origData, origData)
	return base64.StdEncoding.EncodeToString(origData)
}

//解密
func AesDecrypt(cryted string, key string) string {
	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)

	block, err := aes.NewCipher(k)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	blockMode := cipher.NewCBCDecrypter(block, k)
	blockMode.CryptBlocks(crytedByte, crytedByte)
	crytedByte = unpadding(crytedByte)
	return string(crytedByte)
}

// 填充数据
func padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

// 去掉填充数据
func unpadding(src []byte) []byte {
	n := len(src)
	unPadNum := int(src[n-1])
	return src[:n-unPadNum]
}
