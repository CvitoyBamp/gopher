package customerror

import "errors"

var (
	ErrNotEnoughMoney = errors.New("not enough money")
)
