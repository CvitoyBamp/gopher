package customerror

import "errors"

var (
	ErrNotEnoughMoney = errors.New("Not enough money")
)
