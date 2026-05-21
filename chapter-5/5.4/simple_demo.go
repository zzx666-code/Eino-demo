package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

func main() {
	// 创建一个缓冲区为 2 的消息流管道
	reader, writer := schema.Pipe[*schema.Message](2)

	// 在 goroutine 中异步写入数据
	go func() {
		defer writer.Close()
		writer.Send(schema.AssistantMessage("Go语言", nil), nil)
		writer.Send(schema.AssistantMessage("是一门", nil), nil)
		writer.Send(schema.AssistantMessage("高效的", nil), nil)
		writer.Send(schema.AssistantMessage("编程语言。", nil), nil)
	}()

	// 在主 goroutine 中消费流
	for {
		chunk, err := reader.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\n流读取完毕")
			break
		}
		if err != nil {
			fmt.Printf("读取错误: %v\n", err)
			break
		}
		fmt.Print(chunk.Content, ",")
	}
	reader.Close()
}
