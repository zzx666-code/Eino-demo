package main

import (
	"context"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"log"
)

// ReflectionAgent 反思Agent系统
type ReflectionAgent struct {
	client   *openai.Client
	maxRound int // 最大反思轮次
}

func NewReflectionAgent(maxRound int) *ReflectionAgent {
	config := openai.DefaultConfig("sk-95eec8f256f34cb188a42e51c8e0e200")
	config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	config.APIType = openai.APITypeOpenAI
	client := openai.NewClientWithConfig(config)
	return &ReflectionAgent{
		client:   client,
		maxRound: maxRound,
	}
}

// Generate 生成者：根据任务和反馈生成/改进方案
func (r *ReflectionAgent) Generate(ctx context.Context, task string, feedback string) (string, error) {
	userContent := "请完成以下任务：" + task
	if feedback != "" {
		userContent += "\n\n上一轮的反馈意见如下，请据此改进你的方案：\n" + feedback
	}

	resp, err := r.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen3.6-plus",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `你是一个Go语言专家。请认真完成用户的编程任务，给出高质量的代码和设计方案。
如果收到反馈，请仔细理解每条意见并在新方案中逐一改进。`,
			},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature: 0.5,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// Critique 批评者：审查方案并给出改进建议
func (r *ReflectionAgent) Critique(ctx context.Context, task string, solution string) (string, bool, error) {
	resp, err := r.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen-plus",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `你是一个严格的代码审查专家。你的任务是审查给定方案的质量，找出问题和不足。
审查维度：代码正确性、性能、错误处理、可读性、Go最佳实践。

如果方案已经足够好，没有明显问题需要修改，请回复"LGTM"（Looks Good To Me）。
如果有改进空间，请列出具体的问题和改进建议，不要泛泛而谈。`,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("任务：%s\n\n待审查方案：\n%s", task, solution),
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return "", false, err
	}

	feedback := resp.Choices[0].Message.Content
	// 判断是否通过审查
	approved := len(feedback) < 50 || containsApproval(feedback)
	return feedback, approved, nil
}

func containsApproval(s string) bool {
	approvalKeywords := []string{"LGTM", "没有明显问题", "方案已经足够好", "质量很高", "无需修改"}
	for _, kw := range approvalKeywords {
		if contains(s, kw) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Run 执行反思循环
func (r *ReflectionAgent) Run(task string) string {
	ctx := context.Background()
	var feedback string
	var solution string

	for round := 1; round <= r.maxRound; round++ {
		fmt.Printf("\n=== 第%d轮 ===\n", round)

		// Generator 生成/改进方案
		fmt.Println("📝 Generator 正在生成方案...")
		var err error
		solution, err = r.Generate(ctx, task, feedback)
		if err != nil {
			log.Printf("生成失败: %v", err)
			continue
		}
		fmt.Printf("方案长度：%d 字符\n", len(solution))

		// Critic 审查方案
		fmt.Println("🔍 Critic 正在审查...")
		var approved bool
		feedback, approved, err = r.Critique(ctx, task, solution)
		if err != nil {
			log.Printf("审查失败: %v", err)
			continue
		}

		if approved {
			fmt.Println("✅ Critic: LGTM! 方案通过审查")
			return solution
		}

		fmt.Printf("💬 Critic 反馈：%s\n", truncateStr(feedback, 200))
	}

	fmt.Println("⚠️ 达到最大轮次，返回最后一版方案")
	return solution
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func main() {
	agent := NewReflectionAgent(3)
	task := "用Go实现一个线程安全的LRU缓存，支持Get、Put操作和过期时间"

	fmt.Println("任务：", task)
	result := agent.Run(task)
	fmt.Printf("\n=== 最终方案 ===\n%s\n", truncateStr(result, 500))
}
