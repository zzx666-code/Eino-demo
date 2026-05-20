package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen3.6-plus",
	})
	if err != nil {
		panic(err)
	}
	// 分类器：用 Lambda 做简单的关键词分类
	classifier := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		role := input["question"].(string)
		if strings.Contains(role, "react") || strings.Contains(role, " vue") || strings.Contains(role, "html") {
			input["category"] = "front"
		} else if strings.Contains(role, "数据库") || strings.Contains(role, "超时") || strings.Contains(role, "服务器") {
			input["category"] = "back"
		} else {
			input["category"] = "general"
		}
		return input, nil
	})

	// 三个不同角色的 Prompt 模板
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
	// 定义条件路由
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
	// 测试多个问题
	questions := []string{
		"vue的useEffect和useLayoutEffect有什么区别？",
		"Go的GMP调度模型是怎么工作的？",
		"新手程序员应该先学前端还是后端？",
	}

	for _, q := range questions {
		result, err := runner.Invoke(ctx, map[string]any{"question": q})
		if err != nil {
			log.Printf("错误: %v\n", err)
			continue
		}
		fmt.Printf("问题: %s\n%s\n\n", q, result)
	}
}
