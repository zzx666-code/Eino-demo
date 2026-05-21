package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
	"log"
	"sync"
	"time"
)

// 统计token用量
type TokenTracker struct {
	mu              sync.Mutex
	totalPrompt     int
	totalCompletion int
	totalTokens     int
	callCount       int
}

func (t *TokenTracker) Record(usage *model.TokenUsage) {
	if usage == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callCount++
	t.totalCompletion += usage.CompletionTokens
	t.totalPrompt += usage.PromptTokens
	t.totalTokens += usage.TotalTokens
}
func (t *TokenTracker) Report() {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Printf("\n===== Token 用量统计 =====\n")
	fmt.Printf("调用次数: %d\n", t.callCount)
	fmt.Printf("输入Token总计: %d\n", t.totalPrompt)
	fmt.Printf("输出Token总计: %d\n", t.totalCompletion)
	fmt.Printf("Token总计: %d\n", t.totalTokens)
	fmt.Printf("==========================\n")
}
func main() {
	ctx := context.Background()
	tracker := &TokenTracker{}
	// 实现全局通用回调
	globalHandler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			log.Printf("[TRACE] ▶ %s(%s) 开始执行", info.Name, info.Component)
			return context.WithValue(ctx, "trace_start_"+info.Name, time.Now())
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			log.Printf("[TRACE] ▶ %s(%s) 执行结束", info.Name, info.Component)
			if start, ok := ctx.Value("trace_start_" + info.Name).(time.Time); ok {
				duration := time.Since(start)
				log.Printf("[TRACE] ◀ %s(%s) 执行完成, 耗时: %v", info.Name, info.Component, duration)
				// 如果某个组件耗时超过 5 秒，输出警告
				if duration > 5*time.Second {
					log.Printf("[WARN] ⚠ %s 执行耗时过长: %v", info.Name, duration)
				}
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[ERROR] ❌ %s(%s) 执行失败: %v", info.Name, info.Component, err)
			return ctx
		}).Build()
	// 注册全局回调函数
	callbacks.AppendGlobalHandlers(globalHandler)

	// 注册model的回调
	modelHandler := &callbacksHelper.ModelCallbackHandler{
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			tracker.Record(output.TokenUsage)
			if output.TokenUsage != nil {
				fmt.Printf("[模型结束] Token用量: 输入=%d, 输出=%d, 总计=%d\n",
					output.TokenUsage.PromptTokens,
					output.TokenUsage.CompletionTokens,
					output.TokenUsage.TotalTokens)
			}
			return ctx
		},
	}

	handler := callbacksHelper.NewHandlerHelper().
		ChatModel(modelHandler).
		Handler()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 构建链
	chain := compose.NewChain[string, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个Go语言专家，仅仅使用一句话进行回答。"),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName("消息构建"))
	chain.AppendChatModel(chatModel, compose.WithNodeName("通义千问"))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// 模拟多次调用
	questions := []string{
		"Go的slice和array有什么区别？",
		"解释一下Go的GMP调度模型",
		"什么是context.Context？",
	}

	for i, q := range questions {
		fmt.Printf("\n--- 第 %d 次调用 ---\n", i+1)
		result, err := runnable.Invoke(ctx, q,
			compose.WithCallbacks(handler)) // Token回调作为局部回调传入
		if err != nil {
			log.Printf("调用失败: %v", err)
			continue
		}
		// 只打印前80个字符
		content := result.Content
		if len(content) > 80 {
			content = content[:80] + "..."
		}
		fmt.Printf("回复: %s\n", content)
	}

	tracker.Report()
}
