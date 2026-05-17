package main

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

func main() {
	// 使用 GPT-4 的编码方式
	enc, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		panic(err)
	}

	texts := []string{
		"Hello, how are you?",
		"你好，过的怎么样？",
		"Go is are open-source programming language.",
		"大语言模型的核心概念包括Token、Prompt和Temperature。",
	}

	for _, text := range texts {
		tokens := enc.Encode(text, nil, nil)
		fmt.Printf("文本: %s\n", text)
		fmt.Printf("Token数: %d\n", len(tokens))
		fmt.Printf("Token IDs: %v\n\n", tokens)
	}
}
