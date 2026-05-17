package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// WorkingMemory 工作记忆：任务执行期间的临时状态存储
type WorkingMemory struct {
	mu     sync.RWMutex
	store  map[string]interface{}
	taskID string
}

func NewWorkingMemory(taskID string) *WorkingMemory {
	return &WorkingMemory{
		store:  make(map[string]interface{}),
		taskID: taskID,
	}
}

func (w *WorkingMemory) Set(key string, value interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.store[key] = value
	fmt.Printf("  💾 [工作记忆] 写入 %s = %v\n", key, summarize(value))
}

func (w *WorkingMemory) Get(key string) (interface{}, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	val, ok := w.store[key]
	return val, ok
}

func (w *WorkingMemory) GetString(key string) string {
	val, ok := w.Get(key)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	b, _ := json.Marshal(val)
	return string(b)
}

// Snapshot 获取当前工作记忆的快照（注入Prompt用）
func (w *WorkingMemory) Snapshot() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if len(w.store) == 0 {
		return "（工作记忆为空）"
	}
	result := fmt.Sprintf("任务 [%s] 的当前工作状态：\n", w.taskID)
	for k, v := range w.store {
		result += fmt.Sprintf("- %s: %v\n", k, summarize(v))
	}
	return result
}

// Clear 任务完成后清理, 记忆清除
func (w *WorkingMemory) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.store = make(map[string]interface{})
	fmt.Printf("  🧹 [工作记忆] 任务 %s 的工作记忆已清理\n", w.taskID)
}

// summarize 用于将任意值转换为字符串摘要
// 如果转换后的字符串长度超过80个字符，则截取前80个字符并追加省略号
// 使用rune切片确保对多字节字符（如中文）的正确处理
func summarize(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	runes := []rune(s)
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return s
}

// TaskAgent 一个带工作记忆的任务执行Agent
type TaskAgent struct {
	client *openai.Client
	memory *WorkingMemory
}

func NewTaskAgent(taskID string) *TaskAgent {
	config := openai.DefaultConfig(os.Getenv("DASHSCOPE_API_KEY"))
	config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	config.APIType = openai.APITypeOpenAI
	return &TaskAgent{
		client: openai.NewClientWithConfig(config),
		memory: NewWorkingMemory(taskID),
	}
}

// Step1 搜集信息
func (a *TaskAgent) Step1_Collect(ctx context.Context) error {
	fmt.Println("\n--- 步骤1：搜集竞品信息 ---")

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen-plus",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "请列出Go语言Web框架领域的3个主流框架的名称和核心特点，用JSON数组格式返回，每个元素包含name和features字段。",
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return err
	}

	result := resp.Choices[0].Message.Content
	a.memory.Set("competitors", result)
	a.memory.Set("step1_status", "completed")
	a.memory.Set("step1_time", time.Now().Format("15:04:05"))
	return nil
}

// Step2 对比分析（依赖Step1的结果）
func (a *TaskAgent) Step2_Analyze(ctx context.Context) error {
	fmt.Println("\n--- 步骤2：对比分析 ---")

	competitors := a.memory.GetString("competitors")
	if competitors == "" {
		return fmt.Errorf("缺少前置数据：competitors")
	}

	// 将工作记忆中的前序结果注入到Prompt中
	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen-plus",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "你是一个技术分析专家。请基于提供的数据进行对比分析。\n\n当前工作状态：\n" +
					a.memory.Snapshot(),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "基于已收集的竞品信息，请从性能、生态、学习曲线三个维度做一个简要对比分析（100字以内）。",
			},
		},
		Temperature: 0.5,
	})
	if err != nil {
		return err
	}

	analysis := resp.Choices[0].Message.Content
	a.memory.Set("analysis", analysis)
	a.memory.Set("step2_status", "completed")
	return nil
}

// Step3 生成结论（依赖Step1和Step2）
func (a *TaskAgent) Step3_Conclude(ctx context.Context) error {
	fmt.Println("\n--- 步骤3：生成结论 ---")

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen-plus",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "你是一个技术顾问。请基于前序分析给出最终建议。\n\n当前工作状态：\n" +
					a.memory.Snapshot(),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "基于以上信息，用一句话给出你的框架选型建议。",
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return err
	}

	conclusion := resp.Choices[0].Message.Content
	a.memory.Set("conclusion", conclusion)
	a.memory.Set("step3_status", "completed")

	fmt.Printf("\n📋 最终结论：%s\n", conclusion)
	return nil
}

func main() {
	agent := NewTaskAgent("framework-comparison-001")
	ctx := context.Background()

	// 按顺序执行三步任务
	if err := agent.Step1_Collect(ctx); err != nil {
		fmt.Printf("步骤1失败: %v\n", err)
		return
	}
	if err := agent.Step2_Analyze(ctx); err != nil {
		fmt.Printf("步骤2失败: %v\n", err)
		return
	}
	if err := agent.Step3_Conclude(ctx); err != nil {
		fmt.Printf("步骤3失败: %v\n", err)
		return
	}

	// 查看最终的工作记忆快照
	fmt.Println("\n=== 工作记忆最终快照 ===")
	fmt.Println(agent.memory.Snapshot())

	// 任务完成，清理工作记忆
	agent.memory.Clear()
}
