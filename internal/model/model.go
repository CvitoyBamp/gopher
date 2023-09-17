package model

import (
	"time"
)

const (
	StatusNEW        = "NEW"
	StatusREGISTERED = "REGISTERED"
	StatusINVALID    = "INVALID"
	StatusPROCESSING = "PROCESSING"
	StatusPROCESSED  = "PROCESSED"
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

type Withdrawn struct {
	Userid    string    `json:"userid,omitempty"`
	Orderid   string    `json:"order,omitempty"`
	Sum       int       `json:"sum,omitempty"`
	Timestamp time.Time `json:"processed_at,omitempty"`
}

type Accrual struct {
	Orderid string  `json:"order,omitempty"`
	Status  string  `json:"status,omitempty"`
	Accrual *string `json:"accrual,omitempty"`
}

type Balance struct {
	Userid     string    `json:"userid,omitempty"`
	CurBalance string    `json:"current,omitempty"`
	Withdrawn  string    `json:"withdrawn,omitempty"`
	Timestamp  time.Time `json:"uploaded_at,omitempty"`
}
