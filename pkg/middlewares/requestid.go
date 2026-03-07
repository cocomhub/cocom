package middlewares

import (
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/suixibing/cocom/pkg/clog"
)

func RequestID() gin.HandlerFunc {
	return requestid.New(
		requestid.WithCustomHeaderStrKey(HeaderXRequestID),
		requestid.WithHandler(func(c *gin.Context, rid string) {
			if rid != "" {
				c.Request = c.Request.WithContext(clog.WithTraceID(c.Request.Context(), rid))
			}
		}),
	)
}
