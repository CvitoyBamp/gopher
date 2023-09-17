package server

import (
	"net/http"
	"time"
)

func checkLuhn(orderNum string) bool {
	sum := 0
	nDigits := len(orderNum)
	parity := nDigits % 2
	for i := 0; i < nDigits-1; i++ {
		digit := int(orderNum[i])
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum = sum + digit
	}

	return sum%10 == 0
}

func setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(2 * time.Hour),
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    token,
	})
}
