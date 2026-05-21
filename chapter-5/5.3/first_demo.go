package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 创建模型
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}
	//lambda := compose.InvokableLambda(func(ctx context.Context, input string) (output *schema.Message, err error) {
	//	return &schema.Message{
	//		Role:   schema.System,
	//		Content: "你是一个专业的技术助手，回答简洁明了。",
	//	}, nil
	//})
	// 构建一条简单的 Chain：Lambda预处理 → 模型调用
	chain := compose.NewChain[string, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个专业的技术助手，一句话回答问题。"),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName("消息构建")).AppendChatModel(model, compose.WithNodeName("通义千问"))

	// 构建通用回调处理器
	handler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			log.Printf("[开始] 组件=%s 名称=%s 类型=%s", info.Component, info.Name, info.Type)
			// 把开始时间存到 context 中，供 OnEnd 计算耗时
			return context.WithValue(ctx, "start_time_"+info.Name, time.Now())
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if start, ok := ctx.Value("start_time_" + info.Name).(time.Time); ok {
				log.Printf("[结束] 组件=%s 名称=%s 耗时=%v", info.Component, info.Name, time.Since(start))
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[错误] 组件=%s 名称=%s 错误=%v", info.Component, info.Name, err)
			return ctx
		}).
		Build()

	// 编译并运行，通过 WithCallbacks 注入回调
	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	result, err := runnable.Invoke(ctx, "Go语言的channel有什么用？",
		compose.WithCallbacks(handler))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n模型回复:", result.Content)
}
