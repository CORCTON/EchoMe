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
