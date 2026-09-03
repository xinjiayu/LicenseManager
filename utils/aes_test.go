package utils

import (
	"strings"
	"testing"
)

func TestAesEncryptDecryptRoundTrip(t *testing.T) {
	key := "1234567890123456"
	cases := []string{
		"",
		"a",
		"正好十六字节的内容!",
		`{"AppName":"TestApp","ObjUUID":"some-device-uuid"}`,
		strings.Repeat("x", 100),
	}

	for _, plain := range cases {
		encrypted := AesEncrypt(plain, key)
		if encrypted == plain {
			t.Fatalf("加密后内容不应与明文相同")
		}
		decrypted := AesDecrypt(encrypted, key)
		if decrypted != plain {
			t.Fatalf("解密结果与明文不一致:\n期望: %q\n实际: %q", plain, decrypted)
		}
	}
}

// TestAesDecryptRandomIV 每次加密使用随机IV，两次加密结果应不同但都可解密
func TestAesDecryptRandomIV(t *testing.T) {
	key := "1234567890123456"
	plain := "same plaintext"

	first := AesEncrypt(plain, key)
	second := AesEncrypt(plain, key)
	if first == second {
		t.Fatal("随机IV下两次加密结果不应相同")
	}

	if AesDecrypt(first, key) != plain || AesDecrypt(second, key) != plain {
		t.Fatal("两次加密结果都应能正确解密")
	}
}

func TestAesDecryptWrongKey(t *testing.T) {
	encrypted := AesEncrypt("secret content", "1234567890123456")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("错误密钥解密应panic（由上层转换为错误）")
		}
	}()
	AesDecrypt(encrypted, "another-key-12345")
}
