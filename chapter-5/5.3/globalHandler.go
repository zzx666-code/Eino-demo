package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"log"
)

func main() {
	ctx := context.Background()

	// 注册全局回调——对所有编排运行自动生效
	// 使用全局回调
	globalHandler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			log.Printf("[全局] 组件启动: %s(%s)", info.Name, info.Component)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			log.Printf("[全局] 组件完成: %s(%s)", info.Name, info.Component)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[全局告警] 组件异常: %s(%s), 错误: %v", info.Name, info.Component, err)
			return ctx
		}).
		Build()
	callbacks.AppendGlobalHandlers(globalHandler)

	// 创建模型和编排
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	chain := compose.NewChain[string, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个Go语言助手。"),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName("消息构建"))
	chain.AppendChatModel(chatModel, compose.WithNodeName("通义千问"))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 第一次调用——全局回调自动生效，不需要传 WithCallbacks
	fmt.Println("=== 第一次调用 ===")
	result1, err := runnable.Invoke(ctx, "什么是interface？")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("回复:", result1.Content[:50], "...")

	// 第二次调用——全局回调依然生效
	fmt.Println("\n=== 第二次调用 ===")
	result2, err := runnable.Invoke(ctx, "什么是defer？")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("回复:", result2.Content[:50], "...")

	// 第三次调用——同时使用全局回调 + 局部回调
	localHandler := callbacks.NewHandlerBuilder().
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			log.Printf("[局部] 额外的结束处理: %s", info.Name)
			return ctx
		}).
		Build()

	fmt.Println("\n=== 第三次调用（全局+局部） ===")
	result3, err := runnable.Invoke(ctx, "什么是select？",
		compose.WithCallbacks(localHandler))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("回复:", result3.Content[:50], "...")
}
