package response

import "github.com/gin-gonic/gin"

type Response struct {
	Code  string `json:"code"`
	Data  any    `json:"data"`
	Error any    `json:"error"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Response{
		Code:  "OK",
		Data:  data,
		Error: nil,
	})
}

func Err(c *gin.Context, status int, code string, msg string) {
	c.JSON(status, Response{
		Code:  code,
		Data:  nil,
		Error: msg,
	})
}
