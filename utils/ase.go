package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AesEncrypt 加密函数 - 使用随机IV提高安全性
func AesEncrypt(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)

	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("创建AES密码块失败: %v", err))
	}

	// 使用PKCS7填充
	origData = padding(origData, aes.BlockSize)

	// 生成随机IV
	ciphertext := make([]byte, aes.BlockSize+len(origData))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(fmt.Sprintf("生成随机IV失败: %v", err))
	}

	// 加密
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(ciphertext[aes.BlockSize:], origData)

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// AesDecrypt 解密函数 - 支持新格式（随机IV）和旧格式（固定IV）的向后兼容
func AesDecrypt(cryted string, key string) string {
	// 转成字节数组
	crytedByte, err := base64.StdEncoding.DecodeString(cryted)
	if err != nil {
		panic(fmt.Sprintf("Base64解码失败: %v", err))
	}

	k := []byte(key)

	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("创建AES密码块失败: %v", err))
	}

	// 首先尝试新格式（有IV前缀）
	if len(crytedByte) >= aes.BlockSize {
		// 尝试新格式解密
		iv := crytedByte[:aes.BlockSize]
		data := crytedByte[aes.BlockSize:]

		if len(data) > 0 && len(data)%aes.BlockSize == 0 {
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)

			blockMode := cipher.NewCBCDecrypter(block, iv)
			blockMode.CryptBlocks(dataCopy, dataCopy)

			// 尝试去除填充
			if result := tryUnpadding(dataCopy); result != nil {
				return string(result)
			}
		}
	}

	// 如果新格式失败，尝试旧格式（使用密钥作为IV）
	if len(crytedByte) >= aes.BlockSize && len(crytedByte)%aes.BlockSize == 0 {
		// 使用密钥作为IV（旧格式兼容）
		iv := k[:aes.BlockSize] // 使用密钥的前16字节作为IV
		dataCopy := make([]byte, len(crytedByte))
		copy(dataCopy, crytedByte)

		blockMode := cipher.NewCBCDecrypter(block, iv)
		blockMode.CryptBlocks(dataCopy, dataCopy)

		// 尝试去除填充
		if result := tryUnpadding(dataCopy); result != nil {
			return string(result)
		}
	}

	panic("解密失败：数据格式不正确或密钥错误")
}

// tryUnpadding 安全地尝试去除填充，如果失败返回nil
func tryUnpadding(src []byte) []byte {
	defer func() {
		// 捕获panic，表示填充无效
		recover()
	}()

	if len(src) == 0 {
		return nil
	}

	n := len(src)
	unPadNum := int(src[n-1])

	if unPadNum > n || unPadNum == 0 {
		return nil
	}

	// 验证填充的有效性
	for i := n - unPadNum; i < n; i++ {
		if src[i] != byte(unPadNum) {
			return nil
		}
	}

	return src[:n-unPadNum]
}

// padding 填充数据 - PKCS7填充
func padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

// unpadding 去掉填充数据 - PKCS7去填充（严格版本）
func unpadding(src []byte) []byte {
	if len(src) == 0 {
		panic("无效的填充数据")
	}

	n := len(src)
	unPadNum := int(src[n-1])

	if unPadNum > n || unPadNum == 0 {
		panic("无效的填充长度")
	}

	// 验证填充的有效性
	for i := n - unPadNum; i < n; i++ {
		if src[i] != byte(unPadNum) {
			panic("无效的填充数据")
		}
	}

	return src[:n-unPadNum]
}
