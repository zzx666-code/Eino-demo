package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ============================================================
// 与 "Comprehensive actual combat.go" 的核心区别：
//
// 1. [关键词分类器] vs [LLM 分类器]
//    原版：用 strings.Contains 做硬编码关键词匹配
//    本版：用独立的 LLM 做语义理解分类，输出 JSON 结构化结果
//
// 2. 分类器输入/输出
//    原版：匹配固定关键词 → 固定类别（无法理解语义）
//    本版：LLM 理解问题语义 → 输出 {category, reason}（更灵活）
//
// 3. 容错机制
//    原版：匹配不到关键词 → 默认归为 general
//    本版：LLM 解析失败 → 回退到关键词匹配（降级兜底）
//
// 4. 模型选择
//    原版：只有 1 个模型（回答用）
//    本版：2 个模型（轻量模型分类 + 强模型回答）
// ============================================================

// ClassificationResult LLM 分类器的结构化输出
type ClassificationResult struct {
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

func main() {
	ctx := context.Background()

	// ---- 主模型：用于回答问题 ----
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen3.6-plus",
	})
	if err != nil {
		panic(err)
	}

	// ---- 分类专用模型：用更轻量的模型做分类，降低延迟和成本 ----
	classifierModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-turbo",
	})
	if err != nil {
		panic(err)
	}

	// ============================================================
	// ★ 核心改动：LLM 意图分类器（替代关键词匹配）
	// ============================================================
	classifier := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		question := input["question"].(string)

		// 构造分类 Prompt，要求 JSON 结构化输出
		classifyPrompt := fmt.Sprintf(
			`你是一个意图分类器。请判断以下用户问题属于哪个类别，严格以 JSON 格式输出。

可选类别：
- "front"：前端开发相关问题（React、Vue、HTML、CSS、JavaScript、TypeScript、前端框架等）
- "back"：后端开发或运维相关问题（数据库、服务器、Docker、K8s、Linux、API、微服务、网络等）
- "general"：其他一般性问题

请严格按照以下 JSON 格式输出，不要输出任何其他内容：
{"category": "front", "reason": "简要分类原因"}

用户问题：%s`, question)

		messages := []*schema.Message{
			schema.SystemMessage("你是一个专业的意图分类器，只输出 JSON 格式的分类结果，不要包含任何其他文字。"),
			schema.UserMessage(classifyPrompt),
		}

		// 调用轻量 LLM 进行分类
		result, err := classifierModel.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("分类器 LLM 调用失败: %w", err)
		}

		// ---- 解析 LLM 输出 ----
		content := stripThinkTags(result.Content)

		var classResult ClassificationResult
		if err := json.Unmarshal([]byte(content), &classResult); err != nil {
			// ★ 降级兜底：LLM 解析失败时回退到关键词匹配
			log.Printf("⚠ LLM 分类器 JSON 解析失败，回退到关键词匹配: %v", err)
			input["category"] = keywordFallback(question)
			return input, nil
		}

		log.Printf("✓ 分类结果: category=%s, reason=%s", classResult.Category, classResult.Reason)
		input["category"] = classResult.Category
		return input, nil
	})

	// 三个不同角色的 Prompt 模板（与原版相同）
	frontTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个资深前端开发者，擅长代码审查和问题排查，请回答用户的编程问题。"),
		schema.UserMessage("{question}"),
	)

	backTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个工作十年的后端开发人员，精通Linux、Docker和K8s，请回答用户的运维问题。"),
		schema.UserMessage("{question}"),
	)

	generalTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个友好的技术助手，请简洁地回答用户的问题。"),
		schema.UserMessage("{question}"),
	)

	graph := compose.NewGraph[map[string]any, string]()

	_ = graph.AddLambdaNode("classifier", classifier)
	_ = graph.AddChatTemplateNode("front_tpl", frontTpl)
	_ = graph.AddChatTemplateNode("back_tpl", backTpl)
	_ = graph.AddChatTemplateNode("general_tpl", generalTpl)
	_ = graph.AddChatModelNode("model", chatModel)
	_ = graph.AddLambdaNode("formatter", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (string, error) {
			return fmt.Sprintf("[AI助手] %s", msg.Content), nil
		},
	))

	// 条件路由（与原版相同）
	_ = graph.AddEdge(compose.START, "classifier")
	_ = graph.AddBranch("classifier", compose.NewGraphBranch(
		func(ctx context.Context, input map[string]any) (string, error) {
			category := input["category"].(string)
			return category + "_tpl", nil
		},
		map[string]bool{
			"front_tpl":   true,
			"back_tpl":    true,
			"general_tpl": true,
		},
	))
	_ = graph.AddEdge("front_tpl", "model")
	_ = graph.AddEdge("back_tpl", "model")
	_ = graph.AddEdge("general_tpl", "model")
	_ = graph.AddEdge("model", "formatter")
	_ = graph.AddEdge("formatter", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	// 测试用例：包含原版能处理的 + 原版处理不好的边界案例
	questions := []string{
		"vue的useEffect和useLayoutEffect有什么区别？",
		"Go的GMP调度模型是怎么工作的？",
		"新手程序员应该先学前端还是后端？",
		// ★ 以下问题原版关键词分类器处理不好，LLM 分类器可以正确处理：
		"为什么我的网页加载速度很慢？",                  // 语义上是前端，但没有关键词
		"如何优化 MySQL 的慢查询？",                     // 语义上是后端，没有关键词"数据库"
		"K8s 里面 Pod 一直 CrashLoopBackOff 怎么排查？", // 后端，没有关键词"服务器"
		"TypeScript 的泛型约束怎么写？",                 // 前端，没有关键词
	}

	fmt.Println("========================================")
	fmt.Println("  LLM 意图路由系统 - 生产级版本")
	fmt.Println("========================================\n")

	for _, q := range questions {
		result, err := runner.Invoke(ctx, map[string]any{"question": q})
		if err != nil {
			log.Printf("错误: %v\n", err)
			continue
		}
		fmt.Printf("问题: %s\n%s\n\n", q, result)
	}
}

// stripThinkTags 处理 qwen3 等模型的 <think> 推理标签
func stripThinkTags(content string) string {
	if idx := strings.Index(content, "</think>"); idx != -1 {
		content = content[idx+len("</think>"):]
	}
	content = strings.TrimSpace(content)
	// 处理可能的 markdown 代码块包裹
	content = strings.TrimSpace(strings.TrimPrefix(content, "```json"))
	content = strings.TrimSpace(strings.TrimPrefix(content, "```"))
	content = strings.TrimSpace(strings.TrimSuffix(content, "```"))
	return strings.TrimSpace(content)
}

// keywordFallback 关键词匹配降级策略（与原版逻辑一致）
func keywordFallback(question string) string {
	q := strings.ToLower(question)
	if strings.Contains(q, "react") || strings.Contains(q, "vue") || strings.Contains(q, "html") ||
		strings.Contains(q, "css") || strings.Contains(q, "javascript") || strings.Contains(q, "typescript") {
		return "front"
	}
	if strings.Contains(q, "数据库") || strings.Contains(q, "超时") || strings.Contains(q, "服务器") ||
		strings.Contains(q, "docker") || strings.Contains(q, "k8s") || strings.Contains(q, "mysql") ||
		strings.Contains(q, "linux") || strings.Contains(q, "api") || strings.Contains(q, "pod") {
		return "back"
	}
	return "general"
}
