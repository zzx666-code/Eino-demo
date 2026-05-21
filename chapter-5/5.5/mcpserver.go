package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建 MCP Server，声明名称和版本
	s := server.NewMCPServer(
		"DevToolbox",
		"1.0.0",
		server.WithToolCapabilities(true), // 声明支持 Tools 能力
	)

	// 注册字符串处理工具
	s.AddTool(
		mcp.NewTool("string_transform",
			mcp.WithDescription("字符串处理工具，支持大小写转换和字数统计"),
			mcp.WithString("text",
				mcp.Required(),
				mcp.Description("要处理的文本内容"),
			),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Description("操作类型"),
				mcp.Enum("to_upper", "to_lower", "count_chars", "reverse"),
			),
		),
		handleStringTransform,
	)

	// 注册时间工具
	s.AddTool(
		mcp.NewTool("time_util",
			mcp.WithDescription("时间工具，支持获取当前时间和计算日期差"),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Description("操作类型"),
				mcp.Enum("now", "diff"),
			),
			mcp.WithString("format",
				mcp.Description("时间格式，默认为 2006-01-02 15:04:05"),
			),
			mcp.WithString("date1",
				mcp.Description("第一个日期，格式 2006-01-02，计算日期差时必填"),
			),
			mcp.WithString("date2",
				mcp.Description("第二个日期，格式 2006-01-02，计算日期差时必填"),
			),
		),
		handleTimeUtil,
	)

	// 以 SSE 方式启动 Server
	sseServer := server.NewSSEServer(s,
		server.WithBaseURL("http://localhost:8080"),
	)
	fmt.Println("MCP Server (DevToolbox) 启动中，监听 :8080 ...")
	if err := sseServer.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}

// 字符串处理工具的 handler
func handleStringTransform(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")
	operation := req.GetString("operation", "")

	var result string
	switch operation {
	case "to_upper":
		result = strings.ToUpper(text)
	case "to_lower":
		result = strings.ToLower(text)
	case "count_chars":
		count := utf8.RuneCountInString(text)
		result = fmt.Sprintf("文本共 %d 个字符", count)
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)
	default:
		return mcp.NewToolResultError("不支持的操作类型: " + operation), nil
	}

	return mcp.NewToolResultText(result), nil
}

// 时间工具的 handler
func handleTimeUtil(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	operation := req.GetString("operation", "")
	format := req.GetString("format", "2006-01-02 15:04:05")

	switch operation {
	case "now":
		return mcp.NewToolResultText(time.Now().Format(format)), nil
	case "diff":
		date1Str := req.GetString("date1", "")
		date2Str := req.GetString("date2", "")
		if date1Str == "" || date2Str == "" {
			return mcp.NewToolResultError("计算日期差需要提供 date1 和 date2 参数"), nil
		}
		d1, err := time.Parse("2006-01-02", date1Str)
		if err != nil {
			return mcp.NewToolResultError("date1 格式错误，请使用 2006-01-02 格式"), nil
		}
		d2, err := time.Parse("2006-01-02", date2Str)
		if err != nil {
			return mcp.NewToolResultError("date2 格式错误，请使用 2006-01-02 格式"), nil
		}
		diff := d2.Sub(d1)
		days := int(diff.Hours() / 24)
		return mcp.NewToolResultText(fmt.Sprintf("%s 到 %s 相差 %d 天", date1Str, date2Str, days)), nil
	default:
		return mcp.NewToolResultError("不支持的操作类型: " + operation), nil
	}
}
