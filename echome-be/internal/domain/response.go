package domain

import (
	"github.com/labstack/echo/v4"
)

// APIResponse 标准API响应格式
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError 错误信息
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Success 返回成功响应
func Success(c echo.Context, data any) error {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	return c.JSON(200, response)
}

// Created 返回201创建成功响应
func Created(c echo.Context, data any) error {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	return c.JSON(201, response)
}

// Error 返回错误响应
func Error(c echo.Context, statusCode int, code, message string, details ...string) error {
	apiError := &APIError{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		apiError.Details = details[0]
	}

	response := APIResponse{
		Success: false,
		Error:   apiError,
	}

	return c.JSON(statusCode, response)
}

// BadRequest 返回400错误
func BadRequest(c echo.Context, message string, details ...string) error {
	return Error(c, 400, "BAD_REQUEST", message, details...)
}

// NotFound 返回404错误
func NotFound(c echo.Context, message string, details ...string) error {
	return Error(c, 404, "NOT_FOUND", message, details...)
}

// InternalError 返回500错误
func InternalError(c echo.Context, message string, details ...string) error {
	return Error(c, 500, "INTERNAL_ERROR", message, details...)
}
