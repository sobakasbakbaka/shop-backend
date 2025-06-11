package models

type Checkout struct {
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	City string `json:"city"`
	Street string `json:"street"`
	Phone string `json:"phone"`
}