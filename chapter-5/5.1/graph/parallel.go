package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// 使用场景，多模型投票
func main() {
	ctx := context.Background()

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 两个不同视角的模板
	techTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个技术架构师，请从技术可行性角度分析这个需求，用一两句话概括。"),
		schema.UserMessage("{requirement}"),
	)
	productTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个产品经理，请从用户价值角度分析这个需求，用一两句话概括。"),
		schema.UserMessage("{requirement}"),
	)

	// 构建两条子链
	techChain := compose.NewChain[map[string]any, *schema.Message]()
	techChain.AppendChatTemplate(techTpl).AppendChatModel(model)

	productChain := compose.NewChain[map[string]any, *schema.Message]()
	productChain.AppendChatTemplate(productTpl).AppendChatModel(model)

	// 构建并行节点
	parallel := compose.NewParallel()
	parallel.AddGraph("tech", techChain)
	parallel.AddGraph("product", productChain)

	// 主链：并行执行 → 合并结果
	chain := compose.NewChain[map[string]any, string]()
	chain.
		AppendParallel(parallel).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, results map[string]any) (string, error) {
			techResult := results["tech"].(*schema.Message)
			productResult := results["product"].(*schema.Message)
			return fmt.Sprintf("【技术视角】%s\n\n【产品视角】%s",
				techResult.Content, productResult.Content), nil
		}))

	runner, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	result, err := runner.Invoke(ctx, map[string]any{
		"requirement": "为电商App添加AI智能客服功能",
	})
	if err != nil {
		log.Fatal("运行失败:", err)
	}

	fmt.Println(result)
}
