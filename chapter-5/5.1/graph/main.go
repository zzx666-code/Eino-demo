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
		schema.SystemMessage("你是一个技术文档助手，请根据用户的问题给出清晰的解答。"),
		schema.UserMessage("{question}"),
	)

	// 构建 Graph
	graph := compose.NewGraph[map[string]any, string]()

	// 添加节点
	_ = graph.AddChatTemplateNode("tpl", tpl)
	_ = graph.AddChatModelNode("model", model)
	_ = graph.AddLambdaNode("format", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (string, error) {
			return fmt.Sprintf("=== 回答 ===\n%s", msg.Content), nil
		},
	))

	// 连接边：定义数据流向
	_ = graph.AddEdge(compose.START, "tpl")
	_ = graph.AddEdge("tpl", "model")
	_ = graph.AddEdge("model", "format")
	_ = graph.AddEdge("format", compose.END)

	// 编译并运行
	runner, err := graph.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	result, err := runner.Invoke(ctx, map[string]any{
		"question": "什么是 goroutine 泄漏？怎么排查？",
	})
	if err != nil {
		log.Fatal("运行失败:", err)
	}

	fmt.Println(result)
}
