package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func VerifyTelegramAuth(dataURL []string, botToken string) bool {
    checkDataString := dataURL[0]
    hash := dataURL[1]
    
    // Вариант 1: Прямое вычисление HMAC-SHA256 с токеном бота как ключом
    checkHash1 := hmac.New(sha256.New, []byte(botToken))
    checkHash1.Write([]byte(checkDataString))
    calculatedHash1 := checkHash1.Sum(nil)
    calculatedHashString1 := hex.EncodeToString(calculatedHash1)
    
    // Вариант 2: Сначала HMAC-SHA256(botToken, "WebAppData"), затем HMAC-SHA256(dataCheckString, результат)
    secretKey := hmac.New(sha256.New, []byte(botToken))
    secretKey.Write([]byte("WebAppData"))
    secretKeyBytes := secretKey.Sum(nil)
    checkHash2 := hmac.New(sha256.New, secretKeyBytes)
    checkHash2.Write([]byte(checkDataString))
    calculatedHash2 := checkHash2.Sum(nil)
    calculatedHashString2 := hex.EncodeToString(calculatedHash2)
    
    if calculatedHashString1 == hash {
        return true
    } else if calculatedHashString2 == hash {
        return true
    } else {
        return false
    }
}