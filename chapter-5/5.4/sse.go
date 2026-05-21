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
	"net/http"
)

var runnable compose.Runnable[string, *schema.Message]

func init() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	chain := compose.NewChain[string, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		return []*schema.Message{
			schema.SystemMessage("你是一个专业的Go语言助手。"),
			schema.UserMessage(input),
		}, nil
	}))
	chain.AppendChatModel(chatModel)

	runnable, err = chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	question := r.URL.Query().Get("q")
	if question == "" {
		http.Error(w, "缺少参数 q", http.StatusBadRequest)
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	// 调用 Stream 获取流式输出
	stream, err := runnable.Stream(r.Context(), question)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 发送结束事件
			fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
			flusher.Flush()
			break
		}
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			break
		}

		if chunk.Content != "" {
			// 发送数据事件
			fmt.Fprintf(w, "data: %s\n\n", chunk.Content)
			flusher.Flush()
		}
	}
}

func main() {
	http.HandleFunc("/chat/stream", sseHandler)

	fmt.Println("SSE服务启动，监听 :8080")
	fmt.Println("测试: curl 'http://localhost:8080/chat/stream?q=什么是goroutine'")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
