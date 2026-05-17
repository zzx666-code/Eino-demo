package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// Message 对话消息
type Message struct {
	Role    string
	Content string
}

// TokenEstimate 粗略估算一条消息的Token数（中文约1.5字符/Token）
func TokenEstimate(msg Message) int {
	return len([]rune(msg.Content))*2/3 + 10 // 加10是role和格式开销
}

// --- 策略1：滑动窗口 ---

type SlidingWindowMemory struct {
	messages  []Message
	maxRounds int // 保留最近N轮（一问一答算一轮）
}

func NewSlidingWindowMemory(maxRounds int) *SlidingWindowMemory {
	return &SlidingWindowMemory{maxRounds: maxRounds}
}

func (m *SlidingWindowMemory) Add(msg Message) {
	m.messages = append(m.messages, msg)
}

func (m *SlidingWindowMemory) GetHistory() []Message {
	// 一轮 = 2条消息（user + assistant）
	maxMessages := m.maxRounds * 2
	if len(m.messages) <= maxMessages {
		return m.messages
	}
	// 只保留最近 maxMessages 条
	trimmed := m.messages[len(m.messages)-maxMessages:]
	fmt.Printf("  [滑动窗口] 裁剪：%d条 → %d条（保留最近%d轮）\n",
		len(m.messages), len(trimmed), m.maxRounds)
	return trimmed
}

// --- 策略2：摘要压缩 ---

type SummaryMemory struct {
	messages       []Message
	summary        string // 早期对话的摘要
	maxRecent      int    // 保留最近N条原始消息
	summarizeAfter int    // 超过多少条就触发摘要
	client         *openai.Client
}

func NewSummaryMemory(maxRecent, summarizeAfter int, client *openai.Client) *SummaryMemory {
	return &SummaryMemory{
		maxRecent:      maxRecent,
		summarizeAfter: summarizeAfter,
		client:         client,
	}
}

func (m *SummaryMemory) Add(msg Message) {
	m.messages = append(m.messages, msg)
}

func (m *SummaryMemory) GetHistory() []Message {
	if len(m.messages) <= m.summarizeAfter {
		return m.messages
	}

	// 需要摘要的部分：除最近maxRecent条之外的所有消息
	toSummarize := m.messages[:len(m.messages)-m.maxRecent]
	recent := m.messages[len(m.messages)-m.maxRecent:]

	// 调用大模型生成摘要
	summary := m.generateSummary(toSummarize)
	oldTokens := 0
	for _, msg := range toSummarize {
		oldTokens += TokenEstimate(msg)
	}
	fmt.Printf("  [摘要压缩] %d条早期消息（约%d Token）→ 摘要（约%d Token）\n",
		len(toSummarize), oldTokens, TokenEstimate(Message{Content: summary}))

	// 返回：摘要 + 最近的原始消息
	result := []Message{
		{Role: "system", Content: "以下是之前对话的摘要：\n" + summary},
	}
	result = append(result, recent...)
	return result
}

func (m *SummaryMemory) generateSummary(messages []Message) string {
	// 构建对话文本
	var dialog strings.Builder
	for _, msg := range messages {
		dialog.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
	}

	resp, err := m.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: "qwen-turbo", // 摘要用轻量模型即可
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "请用2-3句话简洁地总结以下对话的关键信息，保留重要的事实、决策和用户偏好，省略闲聊内容。",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: dialog.String(),
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		log.Printf("生成摘要失败: %v", err)
		return "（摘要生成失败）"
	}
	return resp.Choices[0].Message.Content
}

// --- 策略3：基于重要性的选择 ---

type ScoredMessage struct {
	Message    Message
	Importance float64 // 0.0 ~ 1.0
}

type ImportanceMemory struct {
	messages []ScoredMessage
	maxToken int
}

func NewImportanceMemory(maxToken int) *ImportanceMemory {
	return &ImportanceMemory{maxToken: maxToken}
}

func (m *ImportanceMemory) Add(msg Message, importance float64) {
	m.messages = append(m.messages, ScoredMessage{Message: msg, Importance: importance})
}

func (m *ImportanceMemory) GetHistory() []Message {
	totalTokens := 0
	for _, sm := range m.messages {
		totalTokens += TokenEstimate(sm.Message)
	}

	if totalTokens <= m.maxToken {
		result := make([]Message, len(m.messages))
		for i, sm := range m.messages {
			result[i] = sm.Message
		}
		return result
	}

	// Token超限，按重要性从低到高排序，逐个移除低重要性消息
	// 但保持原始顺序——先标记要保留的，再按原顺序输出
	keep := make([]bool, len(m.messages))
	for i := range keep {
		keep[i] = true
	}

	currentTokens := totalTokens
	for currentTokens > m.maxToken {
		// 找到重要性最低的消息
		minIdx := -1
		minScore := 2.0
		for i, sm := range m.messages {
			if keep[i] && sm.Importance < minScore {
				minScore = sm.Importance
				minIdx = i
			}
		}
		if minIdx == -1 {
			break
		}
		keep[minIdx] = false
		currentTokens -= TokenEstimate(m.messages[minIdx].Message)
		fmt.Printf("  [重要性筛选] 移除（%.1f分）: %s\n",
			m.messages[minIdx].Importance,
			truncate(m.messages[minIdx].Message.Content, 40))
	}

	var result []Message
	for i, sm := range m.messages {
		if keep[i] {
			result = append(result, sm.Message)
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func main() {
	config := openai.DefaultConfig(os.Getenv("DASHSCOPE_API_KEY"))
	config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	config.APIType = openai.APITypeOpenAI
	client := openai.NewClientWithConfig(config)

	// 模拟一段对话
	dialog := []Message{
		{Role: "user", Content: "你好，我正在开发一个电商系统"},
		{Role: "assistant", Content: "你好！电商系统是一个很好的项目，我可以帮你。"},
		{Role: "user", Content: "我们用Go 1.22开发，数据库用PostgreSQL"},
		{Role: "assistant", Content: "了解，Go 1.22 + PostgreSQL是很好的技术选型。"},
		{Role: "user", Content: "今天天气真不错"},
		{Role: "assistant", Content: "是呢，希望好天气能带来好心情。"},
		{Role: "user", Content: "帮我设计一下订单模块的数据库表结构"},
		{Role: "assistant", Content: "好的，订单模块通常需要orders、order_items、payments等核心表..."},
		{Role: "user", Content: "记住，所有金额字段必须用decimal类型，不能用float"},
		{Role: "assistant", Content: "明白，金额字段统一使用DECIMAL(10,2)，避免浮点精度问题。"},
		{Role: "user", Content: "现在帮我写一下订单创建的API接口"},
	}

	fmt.Println("=== 策略1：滑动窗口（保留最近3轮）===")
	sw := NewSlidingWindowMemory(3)
	for _, msg := range dialog {
		sw.Add(msg)
	}
	history1 := sw.GetHistory()
	fmt.Printf("  保留 %d 条消息\n\n", len(history1))

	fmt.Println("=== 策略2：摘要压缩（超过6条触发摘要，保留最近4条）===")
	sm := NewSummaryMemory(4, 6, client)
	for _, msg := range dialog {
		sm.Add(msg)
	}
	history2 := sm.GetHistory()
	fmt.Printf("  结果 %d 条消息（含1条摘要）\n\n", len(history2))

	fmt.Println("=== 策略3：基于重要性（Token上限3000）===")
	im := NewImportanceMemory(3000)
	importanceScores := []float64{0.5, 0.3, 0.9, 0.4, 0.1, 0.1, 0.7, 0.6, 0.95, 0.5, 0.8}
	for i, msg := range dialog {
		im.Add(msg, importanceScores[i])
	}
	history3 := im.GetHistory()
	fmt.Printf("  保留 %d 条消息\n", len(history3))
}
