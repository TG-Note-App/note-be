package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// #nosec G101
const botToken = "ТВОЙ_BOT_TOKEN"

// VerifyTelegramAuth verifies the Telegram auth hash
func VerifyTelegramAuth(initData string) (bool, error) {
	dataMap, err := parseInitData(initData)
	if err != nil {
		return false, err
	}

	hash, ok := dataMap["hash"]
	if !ok {
		return false, fmt.Errorf("hash not found")
	}
	delete(dataMap, "hash")

	// Формируем строку из параметров
	var dataStrings []string
	for key, value := range dataMap {
		dataStrings = append(dataStrings, key+"="+value)
	}
	sort.Strings(dataStrings)
	dataCheckString := strings.Join(dataStrings, "\n")

	// Создаём HMAC-SHA256
	secretKey := sha256.Sum256([]byte(botToken))
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	return hash == expectedHash, nil
}

func parseInitData(initData string) (map[string]string, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	dataMap := make(map[string]string)
	for key, value := range values {
		if len(value) > 0 {
			dataMap[key] = value[0]
		}
	}

	return dataMap, nil
}
