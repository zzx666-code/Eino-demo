package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// MemoryRecord 一条记忆记录
type MemoryRecord struct {
	ID        string
	Content   string    // 原始文本
	Embedding []float32 // 向量表示
	Metadata  map[string]string
	CreatedAt time.Time
}

// VectorMemoryStore 基于向量的长期记忆存储
type VectorMemoryStore struct {
	records []MemoryRecord
	client  *openai.Client
}

func NewVectorMemoryStore(client *openai.Client) *VectorMemoryStore {
	return &VectorMemoryStore{client: client}
}

// Store 将一条信息存入长期记忆
func (s *VectorMemoryStore) Store(ctx context.Context, id, content string, metadata map[string]string) error {
	// 调用Embedding模型将文本转为向量
	embedding, err := s.getEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("生成embedding失败: %w", err)
	}

	record := MemoryRecord{
		ID:        id,
		Content:   content,
		Embedding: embedding,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
	s.records = append(s.records, record)
	fmt.Printf("  📝 存入记忆 [%s]: %s\n", id, truncate1(content, 50))
	return nil
}

// Retrieve 根据查询检索最相关的K条记忆
func (s *VectorMemoryStore) Retrieve(ctx context.Context, query string, topK int) ([]MemoryRecord, error) {
	queryEmbedding, err := s.getEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("生成查询embedding失败: %w", err)
	}

	// 计算与所有记忆的余弦相似度
	type scored struct {
		record MemoryRecord
		score  float64
	}
	var results []scored
	for _, r := range s.records {
		sim := cosineSimilarity(queryEmbedding, r.Embedding)
		results = append(results, scored{record: r, score: sim})
	}

	// 按相似度降序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 取TopK
	var topResults []MemoryRecord
	limit := topK
	if limit > len(results) {
		limit = len(results)
	}
	for i := 0; i < limit; i++ {
		topResults = append(topResults, results[i].record)
		fmt.Printf("  🔍 检索到 [%s] 相似度=%.4f: %s\n",
			results[i].record.ID, results[i].score,
			truncate1(results[i].record.Content, 50))
	}
	return topResults, nil
}

// getEmbedding 调用通义千问Embedding模型
func (s *VectorMemoryStore) getEmbedding(ctx context.Context, text string) ([]float32, error) {
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: "gte-rerank-v2",
	})
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
	config := openai.DefaultConfig("sk-95eec8f256f34cb188a42e51c8e0e200")
	config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	config.APIType = openai.APITypeOpenAI
	client := openai.NewClientWithConfig(config)
	ctx := context.Background()

	store := NewVectorMemoryStore(client)

	// 存入一些记忆
	fmt.Println("=== 存储阶段 ===")
	memories := []struct {
		id       string
		content  string
		metadata map[string]string
	}{
		{"m1", "用户是一名Go语言开发者，有5年后端开发经验",
			map[string]string{"type": "user_profile"}},
		{"m2", "项目使用Go 1.22 + PostgreSQL + Redis技术栈",
			map[string]string{"type": "project_info"}},
		{"m3", "用户要求所有金额字段必须使用decimal类型，不能用float",
			map[string]string{"type": "requirement"}},
		{"m4", "订单模块已经设计完成，包含orders、order_items、payments三张核心表",
			map[string]string{"type": "progress"}},
		{"m5", "用户偏好简洁的代码风格，不喜欢过度封装",
			map[string]string{"type": "preference"}},
	}

	for _, m := range memories {
		if err := store.Store(ctx, m.id, m.content, m.metadata); err != nil {
			log.Fatalf("存储失败: %v", err)
		}
	}

	// 检索测试
	fmt.Println("\n=== 检索阶段 ===")

	queries := []string{
		"用户的技术背景是什么",
		"数据库表结构怎么设计的",
		"金额相关的开发规范",
	}

	for _, q := range queries {
		fmt.Printf("\n查询: %s\n", q)
		results, err := store.Retrieve(ctx, q, 2)
		if err != nil {
			log.Printf("检索失败: %v", err)
			continue
		}
		_ = results
	}
}

func truncate1(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
