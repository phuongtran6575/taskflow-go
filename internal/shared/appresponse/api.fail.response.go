package appresponse

import "github.com/gin-gonic/gin"

func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}

// BR-ROLE-03/BR-ROLE-06: Fail with additional data payload alongside error info
func FailWithData(c *gin.Context, status int, code, message string, data interface{}) {
	c.JSON(status, Response{
		Success: false,
		Data:    data,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}
