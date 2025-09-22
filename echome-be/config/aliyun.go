package config

// 阿里云相关的常量和默认值
const (
	// DefaultALBLEndpoint 阿里云百炼服务的默认端点
	DefaultALBLEndpoint = "https://dashscope.aliyuncs.com"
	
	// ALBLServiceType 阿里云百炼服务类型
	ALBLServiceType = "alibailian"
)

// GetALBLApiKey 获取阿里云百炼API密钥
func (c *Config) GetALBLApiKey() string {
	return c.ALBL.APIKey
}

// GetALBLSecret 获取阿里云百炼密钥密码
func (c *Config) GetALBLSecret() string {
	return c.ALBL.APISecret
}

// GetALBLEndpoint 获取阿里云百炼服务端点
func (c *Config) GetALBLEndpoint() string {
	if c.ALBL.Endpoint != "" {
		return c.ALBL.Endpoint
	}
	return DefaultALBLEndpoint
}