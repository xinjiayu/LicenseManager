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

// AesDecrypt 解密函数（兼容入口）：先按新格式（随机IV前缀）解密，失败再按旧格式（密钥作IV）解密。
// 两种格式仅以PKCS7填充校验作为判据，无法区分"错误IV只破坏首块"的误判；
// 调用方若能校验明文结构（如JSON），应使用 AesDecryptPrefixIV / AesDecryptFixedIV 自行判别。
func AesDecrypt(cryted string, key string) string {
	if plain, err := AesDecryptPrefixIV(cryted, key); err == nil {
		return plain
	}
	if plain, err := AesDecryptFixedIV(cryted, key); err == nil {
		return plain
	}
	panic("解密失败：数据格式不正确或密钥错误")
}

// AesDecryptPrefixIV 按新格式解密：密文前16字节为随机IV
func AesDecryptPrefixIV(cryted string, key string) (string, error) {
	crytedByte, err := base64.StdEncoding.DecodeString(cryted)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("创建AES密码块失败: %w", err)
	}

	if len(crytedByte) < 2*aes.BlockSize || len(crytedByte)%aes.BlockSize != 0 {
		return "", fmt.Errorf("密文长度异常: %d", len(crytedByte))
	}

	iv := crytedByte[:aes.BlockSize]
	data := crytedByte[aes.BlockSize:]
	plaintext := cbcDecrypt(block, iv, data)
	if plaintext == nil {
		return "", fmt.Errorf("新格式解密失败：填充校验未通过")
	}
	return string(plaintext), nil
}

// AesDecryptFixedIV 按旧格式解密：以密钥前16字节作为IV（向后兼容）
func AesDecryptFixedIV(cryted string, key string) (string, error) {
	crytedByte, err := base64.StdEncoding.DecodeString(cryted)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}

	k := []byte(key)
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", fmt.Errorf("创建AES密码块失败: %w", err)
	}

	if len(crytedByte) < aes.BlockSize || len(crytedByte)%aes.BlockSize != 0 {
		return "", fmt.Errorf("密文长度异常: %d", len(crytedByte))
	}

	plaintext := cbcDecrypt(block, k[:aes.BlockSize], crytedByte)
	if plaintext == nil {
		return "", fmt.Errorf("旧格式解密失败：填充校验未通过")
	}
	return string(plaintext), nil
}

// cbcDecrypt 用指定IV做CBC解密并去除PKCS7填充，填充非法时返回nil
func cbcDecrypt(block cipher.Block, iv, data []byte) []byte {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dataCopy, dataCopy)
	return tryUnpadding(dataCopy)
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
