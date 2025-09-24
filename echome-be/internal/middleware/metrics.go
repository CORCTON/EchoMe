package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// MetricsMiddleware 添加基本的性能监控中间件
func MetricsMiddleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// 执行下一个处理器
			err := next(c)

			// 计算响应时间
			duration := time.Since(start)

			// 记录基本指标
			c.Logger().Info("Request completed",
				"method", c.Request().Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"duration_ms", duration.Milliseconds(),
				"size", c.Response().Size,
			)

			// 添加响应头
			c.Response().Header().Set("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")

			return err
		}
	})
}
