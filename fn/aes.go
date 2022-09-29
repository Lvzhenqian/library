package fn

import (
	"crypto/aes"
	"crypto/cipher"
)

func AesEncryptCBC(plainText []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	paddingText := pkcs7Padding(plainText)
	encrypted := make([]byte, len(paddingText))
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(encrypted, paddingText)
	return encrypted, nil
}

func AesDecryptCBC(encrypted []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	blockMode.CryptBlocks(decrypted, encrypted)
	return pkcs7UnPadding(decrypted), nil
}

// =================== ECB ======================
func AesEncryptECB(origData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plainText := pkcs7Padding(origData)
	encrypted := make([]byte, len(plainText))
	for bs, be := 0, block.BlockSize(); bs <= len(origData); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Encrypt(encrypted[bs:be], plainText[bs:be])
	}

	return encrypted, nil
}

func AesDecryptECB(encrypted []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	decrypted := make([]byte, len(encrypted))

	for bs, be := 0, block.BlockSize(); bs < len(encrypted); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}
	return pkcs7UnPadding(decrypted), nil
}

// =================== CFB ======================
func AesEncryptCFB(origData []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plainText := pkcs7Padding(origData)
	encrypted := make([]byte, len(plainText))

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(encrypted, plainText)
	return encrypted, nil
}

func AesDecryptCFB(encrypted []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	decrypted := make([]byte, len(encrypted))
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(decrypted, encrypted)
	return pkcs7UnPadding(decrypted), nil
}

func pkcs7Padding(plainText []byte) []byte {
	length := (len(plainText) + aes.BlockSize) / aes.BlockSize
	paddingText := make([]byte, length*aes.BlockSize)
	copy(paddingText, plainText)

	for i := len(plainText); i < len(paddingText); i++ {
		paddingText[i] = byte(len(paddingText) - len(plainText))
	}
	return paddingText
}

func pkcs7UnPadding(plainText []byte) []byte {
	if len(plainText) > 0 {
		trim := len(plainText) - int(plainText[len(plainText)-1])
		return plainText[:trim]
	}
	return plainText
}
