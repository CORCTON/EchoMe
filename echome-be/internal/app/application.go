package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/handler"
	"github.com/justin/echome-be/internal/middleware"
	"github.com/justin/echome-be/internal/validation"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/sync/errgroup"

	// swag 生成的文档
	_ "github.com/justin/echome-be/docs"
)

// Application 应用程序主体
type Application struct {
	config    *config.Config
	handler   *handler.Handlers
	echo      *echo.Echo
	validator *validation.ConfigValidator
}

// NewApplication 初始化应用
func NewApplication(cfg *config.Config, h *handler.Handlers) *Application {
	e := echo.New()

	// 注册中间件
	e.Use(
		echomiddleware.Logger(),
		echomiddleware.Recover(),
		echomiddleware.CORS(),
		middleware.MetricsMiddleware(),
		echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStore(20)), // 简单的内存限流
	)

	// 注册路由
	h.RegisterRoutes(e)

	// swagger 文档路由
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	return &Application{
		config:    cfg,
		handler:   h,
		echo:      e,
		validator: validation.NewConfigValidator(),
	}
}

// GetEcho 获取 Echo 实例（用于测试）
func (a *Application) GetEcho() *echo.Echo {
	return a.echo
}

// Run 启动应用
func (a *Application) Run() error {
	// 捕获系统信号，支持优雅退出
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 使用 errgroup 管理 goroutine
	g, gCtx := errgroup.WithContext(ctx)

	// ---------------------------
	// 1. 启动 HTTP 服务
	// ---------------------------
	g.Go(func() error {
		zap.L().Info("服务器启动中",
			zap.String("端口", a.config.Server.Port),
			zap.String("API文档", fmt.Sprintf("http://localhost:%s/swagger/", a.config.Server.Port)),
			zap.String("健康检查", fmt.Sprintf("http://localhost:%s/health", a.config.Server.Port)),
			zap.String("AI服务", a.config.AI.ServiceType),
		)

		server := &http.Server{
			Addr:    fmt.Sprintf(":%s", a.config.Server.Port),
			Handler: a.echo,
		}

		// 监听退出信号，触发优雅关闭
		go func() {
			<-gCtx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			zap.L().Info("正在关闭服务器...")
			if err := server.Shutdown(shutdownCtx); err != nil {
				zap.L().Error("关闭服务器时出错", zap.Error(err))
			}
		}()

		// 启动服务
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("服务启动失败: %w", err)
		}
		return nil
	})

	// ---------------------------
	// 2. 定时任务：检查角色语音状态
	// ---------------------------
	g.Go(func() error {
		ticker := time.NewTicker(10 * time.Minute) // 每 10 分钟检查一次
		defer ticker.Stop()

		timer := time.NewTimer(5 * time.Second) // 延迟 5 秒后进行首次检查
		defer timer.Stop()

		for {
			select {
			case <-gCtx.Done():
				zap.L().Info("语音状态检查任务已停止")
				return nil
			case <-timer.C: // 首次检查
				zap.L().Info("启动时检查角色语音状态")
				if err := a.handler.GetRouter().GetCharacterService().CheckAndUpdatePendingCharacters(gCtx); err != nil {
					zap.L().Error("启动检查角色语音状态失败", zap.Error(err))
				}
			case <-ticker.C: // 定时检查
				zap.L().Info("定时检查角色语音状态")
				if err := a.handler.GetRouter().GetCharacterService().CheckAndUpdatePendingCharacters(gCtx); err != nil {
					zap.L().Error("定时检查角色语音状态失败", zap.Error(err))
				}
			}
		}
	})

	// 等待所有 goroutine 完成
	if err := g.Wait(); err != nil {
		zap.L().Error("应用运行出错", zap.Error(err))
		return err
	}

	zap.L().Info("服务器已优雅退出")
	return nil
}
