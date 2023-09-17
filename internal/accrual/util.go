package accrual

import (
	"math/rand"
	"strconv"
)

func randNumber(n int) int {
	return rand.Intn(n)
}

func randAccrual() string {
	return strconv.FormatFloat(rand.Float64()*1000, 'f', -1, 64)
}
