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
		log.Fatal(err)
	}

	// 构建 Chain：消息构建 → 模型调用 → 后处理（普通Lambda）
	chain := compose.NewChain[string, string]()

	// 第一个节点：构建消息
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个翻译助手，将中文翻译成英文，只输出翻译结果。"),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName("消息构建"))

	// 第二个节点：模型调用（Stream模式下输出是流）
	chain.AppendChatModel(chatModel, compose.WithNodeName("翻译模型"))

	// 第三个节点：普通Lambda，输入是完整的 *schema.Message
	// 当用 Stream 运行时，框架会自动把模型的流式输出拼接成完整消息，再传给这个Lambda
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		return fmt.Sprintf("翻译结果（共%d个字符）: %s", len(msg.Content), msg.Content), nil
	}), compose.WithNodeName("格式化输出"))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 用 Invoke 调用——一切正常，没有流的问题
	fmt.Println("=== Invoke 模式 ===")
	result, err := runnable.Invoke(ctx, "Go语言是世界上最好的语言")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)

	// 用 Stream 调用——模型会流式输出，但后面的Lambda需要完整输入
	// Eino 自动在模型和Lambda之间插入拼接逻辑
	fmt.Println("\n=== Stream 模式 ===")
	stream, err := runnable.Stream(ctx, "并发是Go语言的核心优势")
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
		fmt.Print(chunk + ",")
	}
	fmt.Println()
}
