package middlewares

import (
	"github.com/CvitoyBamp/gopher/internal/database"
	privateJWT "github.com/CvitoyBamp/gopher/internal/jwt"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/matthewhartstonge/argon2"
	"net/http"
	"strconv"
	"strings"
)

func VerifyMiddleware(db *database.Postgres) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if strings.Contains(r.URL.String(), "register") || strings.Contains(r.URL.String(), "login") {
				if r.Header.Get("Content-Type") != "application/json" {
					http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if r.Header.Get("Authorization") == "" {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			jwtauth.Verifier(privateJWT.TokenAuth)

			token, err := privateJWT.TokenAuth.Decode(jwtauth.TokenFromHeader(r))

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil || jwt.Validate(token) != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			username, password := token.PrivateClaims()["username"], token.PrivateClaims()["password"]

			userID, pass, errDB := db.GetUserData(username.(string))
			if errDB != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ok, errVerify := argon2.VerifyEncoded([]byte(password.(string)), []byte(pass))
			if errVerify != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			r.Header.Set("Gopher-User-Id", strconv.Itoa(userID))

			next.ServeHTTP(w, r)

		})
	}
}
