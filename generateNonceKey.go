package gows

import "crypto/rand"
import "encoding/base64"

func generateNonceKey() (string, error) {

	bytes := make([]byte, 16)

	_, err := rand.Read(bytes)

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil

}
