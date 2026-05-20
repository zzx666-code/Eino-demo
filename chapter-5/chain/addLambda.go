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

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个翻译专家，请将用户输入的中文翻译成英文。"),
		schema.UserMessage("{text}"),
	)

	// 构建 Chain：模板 → 模型 → 后处理
	chain := compose.NewChain[map[string]any, string]()
	chain.
		AppendChatTemplate(tpl).
		AppendChatModel(model).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
			// 自定义后处理：提取内容并格式化
			return fmt.Sprintf("【翻译结果】%s", msg.Content), nil
		}))

	runner, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	result, err := runner.Invoke(ctx, map[string]any{
		"text": "今天天气真不错，适合出去走走。",
	})
	if err != nil {
		log.Fatal("运行失败:", err)
	}

	fmt.Println(result)
}
