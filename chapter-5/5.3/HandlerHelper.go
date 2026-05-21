package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)

func main() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 定义 ChatModel 专用的回调处理器
	modelHandler := &callbacksHelper.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			fmt.Printf("[模型开始] 名称=%s, 输入消息数=%d\n", info.Name, len(input.Messages))
			// ★ 正确方式：通过 context.WithValue 传递开始时间给 OnEnd
			// input.Extra 和 output.Extra 是两个独立的 map，框架不会互相复制
			ctx = context.WithValue(ctx, "start_time", time.Now())
			// 打印最后一条用户消息
			for i := len(input.Messages) - 1; i >= 0; i-- {
				if input.Messages[i].Role == schema.User {
					fmt.Printf("[模型开始] 用户输入: %s\n", input.Messages[i].Content)
					break
				}
			}
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			fmt.Printf("[模型结束] 名称=%s\n", info.Name)
			fmt.Printf("[模型] 名称=%s\n", output.Config.Model)
			// ★ 从 context 中读取 OnStart 存入的开始时间
			if startTime, ok := ctx.Value("start_time").(time.Time); ok {
				fmt.Printf("[模型耗时] %s → 耗时: %v\n", info.Name, time.Since(startTime))
			}
			if output.TokenUsage != nil {
				fmt.Printf("[模型结束] Token用量: 输入=%d, 输出=%d, 总计=%d\n",
					output.TokenUsage.PromptTokens,
					output.TokenUsage.CompletionTokens,
					output.TokenUsage.TotalTokens)
			}
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			fmt.Printf("[模型错误] 名称=%s, 错误=%v\n", info.Name, err)
			return ctx
		},
	}

	// 定义 Tool 专用的回调处理器
	toolHandler := &callbacksHelper.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			fmt.Printf("[工具开始] 名称=%s, 参数=%s\n", info.Name, input.ArgumentsInJSON)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			fmt.Printf("[工具结束] 名称=%s, 结果=%s\n", info.Name, output.Response)
			return ctx
		},
	}

	// 用 HandlerHelper 组合多个组件回调
	handler := callbacksHelper.NewHandlerHelper().
		ChatModel(modelHandler).
		Tool(toolHandler).
		Handler()

	// 构建 Chain
	chain := compose.NewChain[string, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个专业的Go语言助手。"),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName("消息构建"))
	chain.AppendChatModel(chatModel, compose.WithNodeName("通义千问"))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	result, err := runnable.Invoke(ctx, "简单介绍下Go的goroutine",
		compose.WithCallbacks(handler))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n回复:", result.Content)
}
