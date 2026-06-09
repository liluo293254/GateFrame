package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: "ok", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: "ok", Data: data})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Body{Code: code, Message: message})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
}

func Forbidden(c *gin.Context) {
	Error(c, http.StatusForbidden, "forbidden", "forbidden")
}

func NotFound(c *gin.Context) {
	Error(c, http.StatusNotFound, "not_found", "not found")
}

func Internal(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "internal_error", "internal error")
}
