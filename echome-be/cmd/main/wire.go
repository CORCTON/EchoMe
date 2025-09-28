//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/app"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/handler"
	"github.com/justin/echome-be/internal/infra"
)

// InitializeApplication creates a new Application with all dependencies
// 参数: configPath - 配置文件路径
func InitializeApplication(configPath string) (*app.Application, error) {
	panic(wire.Build(
		// Configuration
		config.ConfigProviderSet,
		// Repositories (Infrastructure Layer)
		infra.RepositoryProviderSet,
		// Services (Domain Layer)
		domain.ServiceProviderSet,
		// Handlers
		handler.HandlerProviderSet,
		// Application
		app.NewApplication,
	))
}
