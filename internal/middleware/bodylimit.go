package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes corta la lectura del body en maxBytes; una petición mas grande
// falla al leerla (Gin responde 400 al fallar el bind) en vez de que el
// servidor intente cargarla entera en memoria.
func MaxBodyBytes(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
