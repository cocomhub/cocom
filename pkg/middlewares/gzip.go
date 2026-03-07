package middlewares

import (
	"compress/gzip"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.writer.Write(b)
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

func Gzip() gin.HandlerFunc {
	enabled := viper.GetBool("server.gzip.enabled")
	if !enabled {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		ae := c.GetHeader("Accept-Encoding")
		if !strings.Contains(ae, "gzip") {
			c.Next()
			return
		}
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		c.Writer.Header().Del("Content-Length")
		gz, _ := gzip.NewWriterLevel(c.Writer, gzip.BestSpeed)
		defer gz.Close()
		grw := &gzipResponseWriter{ResponseWriter: c.Writer, writer: gz}
		c.Writer = grw
		c.Next()
	}
}
