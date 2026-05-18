package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool/utils"
)

// 通过 struct tag 定义参数的描述、约束
type SearchRequest struct {
	Query    string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
	MaxCount int    `json:"max_count" jsonschema_description:"最多返回的结果数量，默认5"`
	Language string `json:"language" jsonschema:"enum=zh,enum=en" jsonschema_description:"结果语言，zh为中文，en为英文"`
}

type SearchResult struct {
	Items []SearchItem `json:"items"`
	Total int          `json:"total"`
}

type SearchItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Summary string `json:"summary"`
}

func searchWeb(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	// 模拟搜索逻辑
	maxCount := req.MaxCount
	if maxCount <= 0 {
		maxCount = 5
	}

	items := []SearchItem{
		{Title: "Go语言官方文档", URL: "https://go.dev/doc/", Summary: "Go编程语言官方文档和教程"},
		{Title: "Eino框架指南", URL: "https://cloudwego.io/docs/eino", Summary: "字节跳动开源的Go语言LLM应用开发框架"},
		{Title: "Go并发编程实战", URL: "https://example.com/go-concurrency", Summary: "深入讲解goroutine和channel的用法"},
	}

	// 简单过滤
	filtered := make([]SearchItem, 0)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title+item.Summary), strings.ToLower(req.Query)) {
			filtered = append(filtered, item)
		}
		if len(filtered) >= maxCount {
			break
		}
	}

	return &SearchResult{Items: filtered, Total: len(filtered)}, nil
}

func main() {
	// InferTool 从函数签名和 struct tag 自动推断工具信息
	searchTool, err := utils.InferTool("web_search", "搜索互联网上的信息，返回相关网页的标题、链接和摘要", searchWeb)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 查看自动推断出的工具信息
	info, _ := searchTool.Info(ctx)
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println("自动推断的工具信息：")
	fmt.Println(string(infoJSON))

	fmt.Println()

	// 执行工具
	_ = time.Now() // 占位，实际项目中可能用于日志
	result, err := searchTool.InvokableRun(ctx, `{"query": "Go", "max_count": 2, "language": "zh"}`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("搜索结果：")
	fmt.Println(result)
}
