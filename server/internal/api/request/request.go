package request

import "github.com/gin-gonic/gin"

func GetRequestID(c *gin.Context) string {
	return c.GetString("request_id")
}
