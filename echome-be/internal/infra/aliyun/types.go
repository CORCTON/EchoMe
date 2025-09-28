package aliyun

// TavilySearchRequest 搜索请求参数

type TavilySearchRequest struct {
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth"`
	IncludeAnswer bool   `json:"include_answer"`
	MaxResults    int    `json:"max_results"`
}

// TavilySearchResult 单个搜索结果

type TavilySearchResult struct {
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// TavilySearchResponse 搜索响应结果

type TavilySearchResponse struct {
	Answer  string               `json:"answer"`
	Results []TavilySearchResult `json:"results"`
}


// VoiceCloneRequest 阿里云声音复刻API请求结构
type VoiceCloneRequest struct {
	Model string `json:"model"`
	Input struct {
		Action      string `json:"action"`       // 固定为create_voice
		TargetModel string `json:"target_model"` // 声音复刻使用的模型
		Prefix      string `json:"prefix"`       // 音色自定义前缀
		URL         string `json:"url"`          // 音频文件URL
		VoiceID     string `json:"voice_id"`     // 查询时使用的音色ID
	} `json:"input"`
}

// VoiceCloneAPIResponse 阿里云声音复刻API响应结构
type VoiceCloneAPIResponse struct {
	Output struct {
		VoiceID string `json:"voice_id"`
	} `json:"output"`
	Usage struct {
		Count int `json:"count"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}
