package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 定义模板
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{domain}领域的专家，请用一句话回答。"),
		schema.UserMessage("{question}"),
	)

	// 创建 Chain：模板 → 模型
	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(template)
	chain.AppendChatModel(cm)

	// 编译
	runnable, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 运行：直接传入变量 map，Chain 会自动完成模板格式化和模型调用
	resp, err := runnable.Invoke(ctx, map[string]any{
		"domain":   "分布式系统",
		"question": "CAP 定理是什么？",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}
