package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"io"
	"log"
)

func main() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		return
	}
	// 构建 Graph（输入 string，输出 map[string]any 汇聚两个分支结果）
	g := compose.NewGraph[string, map[string]any]()

	// 节点1：构建翻译消息
	_ = g.AddLambdaNode("build_translate_msg",
		compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
			return []*schema.Message{
				schema.SystemMessage("将以下中文翻译成英文，只输出翻译结果："),
				schema.UserMessage(input),
			}, nil
		}),
		compose.WithNodeName("翻译消息构建"),
	)

	// 节点2：构建摘要消息
	_ = g.AddLambdaNode("build_summary_msg",
		compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
			return []*schema.Message{
				schema.SystemMessage("用一句话概括以下内容的核心观点："),
				schema.UserMessage(input),
			}, nil
		}),
		compose.WithNodeName("摘要消息构建"),
	)

	// 节点3：翻译模型
	_ = g.AddChatModelNode("translate_model", chatModel,
		compose.WithNodeName("翻译模型"),
	)

	// 节点4：摘要模型
	_ = g.AddChatModelNode("summary_model", chatModel,
		compose.WithNodeName("摘要模型"),
	)

	// 节点5：提取翻译结果，使用 WithOutputKey 将输出写入 map 的 "translate" key
	_ = g.AddLambdaNode("extract_translate",
		compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
			return msg.Content, nil
		}),
		compose.WithNodeName("提取翻译"),
		compose.WithOutputKey("translate"),
	)

	// 节点6：提取摘要结果，使用 WithOutputKey 将输出写入 map 的 "summary" key
	_ = g.AddLambdaNode("extract_summary",
		compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
			return msg.Content, nil
		}),
		compose.WithNodeName("提取摘要"),
		compose.WithOutputKey("summary"),
	)

	// 连接边：START 同时连到两个消息构建节点（并行分支）
	_ = g.AddEdge(compose.START, "build_translate_msg")
	_ = g.AddEdge(compose.START, "build_summary_msg")
	_ = g.AddEdge("build_translate_msg", "translate_model")
	_ = g.AddEdge("build_summary_msg", "summary_model")
	_ = g.AddEdge("translate_model", "extract_translate")
	_ = g.AddEdge("summary_model", "extract_summary")
	// 两个分支各自直接连到 END，通过 OutputKey 自动汇聚到 map[string]any
	_ = g.AddEdge("extract_translate", compose.END)
	_ = g.AddEdge("extract_summary", compose.END)

	runnable, err := g.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	input := "Go语言通过goroutine和channel实现了优雅的并发模型，它让开发者不需要直接操作线程和锁，就能写出高性能的并发程序。"

	// 使用 Invoke 同步调用
	fmt.Println("=== Invoke 模式 ===")
	result, err := runnable.Invoke(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	translate, _ := result["translate"].(string)
	summary, _ := result["summary"].(string)
	fmt.Printf("📝 英文翻译:\n%s\n\n📌 中文摘要:\n%s\n", translate, summary)

	// 使用 Stream 流式调用
	fmt.Println("\n=== Stream 模式 ===")
	stream, err := runnable.Stream(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		for key, val := range chunk {
			fmt.Printf("[%s] %v", key, val)
		}
	}
	fmt.Println()
}
