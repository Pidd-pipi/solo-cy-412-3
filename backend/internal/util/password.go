package util

import "golang.org/x/crypto/bcrypt"

func HashPassword(v string) (string, error) {
	b, e := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
	return string(b), e
}
func CheckPassword(hash, v string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(v))
}
