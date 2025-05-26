package utils

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func GetUserID(c *gin.Context) (uint, error) {
	val, ok := c.Get("user_id")
	if !ok {
		return 0, errors.New("user_id not found in context")
	}
	f64, ok := val.(float64)
	if !ok {
		return 0, errors.New("user_id is not a float64")
	}
	return uint(f64), nil
}