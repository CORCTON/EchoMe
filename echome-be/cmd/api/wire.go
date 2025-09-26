//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/client"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/app"
	"github.com/justin/echome-be/internal/infra"
	"github.com/justin/echome-be/internal/interfaces"
	"github.com/justin/echome-be/internal/service"
)

// InitializeApplication creates a new Application with all dependencies
// 参数: configPath - 配置文件路径
func InitializeApplication(configPath string) (*app.Application, error) {
	panic(wire.Build(
		// Configuration
		config.ConfigProviderSet,
		// Repositories (Infrastructure Layer)
		infra.RepositoryProviderSet,
		// AI Services (External Services)
		client.AIServiceProviderSet,
		// Business Services (Service Layer)
		service.ServiceProviderSet,
		// Handlers (Interface Layer)
		interfaces.HandlerProviderSet,
		// Application
		app.NewApplication,
	))
}
