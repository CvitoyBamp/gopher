package model

import (
	"time"
)

type Register struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Order struct {
	Userid    string    `json:"userid,omitempty"`
	Orderid   string    `json:"number,omitempty"`
	Status    string    `json:"status,omitempty"`
	Accrual   *string   `json:"accrual,omitempty"`
	Timestamp time.Time `json:"uploaded_at,omitempty"`
}

type Balance struct {
	Userid     string    `json:"userid,omitempty"`
	CurBalance string    `json:"current,omitempty"`
	Withdrawn  string    `json:"withdrawn,omitempty"`
	Timestamp  time.Time `json:"uploaded_at,omitempty"`
}
