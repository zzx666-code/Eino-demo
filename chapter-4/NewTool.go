package main

//
//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"github.com/cloudwego/eino/components/tool/utils"
//	"github.com/cloudwego/eino/schema"
//	"log"
//	"math"
//)
//
//type WeatherRequest struct {
//	City string `json:"city"`
//}
//
//type WeatherResponse struct {
//	City    string `json:"city"`
//	Temp    string `json:"temp"`
//	Weather string `json:"weather"`
//}
//
//// getWeather 工具就是函数的封装
//func getWeather(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
//	// 这里用硬编码模拟，实际项目中你会去调天气 API
//	mockData := map[string]WeatherResponse{
//		"北京": {City: "北京", Temp: "22°C", Weather: "晴"},
//		"上海": {City: "上海", Temp: "26°C", Weather: "多云"},
//		"深圳": {City: "深圳", Temp: "30°C", Weather: "阵雨"},
//	}
//
//	if data, ok := mockData[req.City]; ok {
//		return &data, nil
//	}
//	return &WeatherResponse{City: req.City, Temp: "未知", Weather: "未知"}, nil
//}
//
//type CalcRequest struct {
//	A  float64 `json:"a"`
//	B  float64 `json:"b"`
//	Op string  `json:"op"`
//}
//
//type CalcResponse struct {
//	Expression string  `json:"expression"`
//	Result     float64 `json:"result"`
//}
//
//func calculate(ctx context.Context, req *CalcRequest) (*CalcResponse, error) {
//	var result float64
//	switch req.Op {
//	case "add":
//		result = req.A + req.B
//	case "subtract":
//		result = req.A - req.B
//	case "multiply":
//		result = req.A * req.B
//	case "divide":
//		if req.B == 0 {
//			return nil, fmt.Errorf("除数不能为零")
//		}
//		result = req.A / req.B
//	default:
//		return nil, fmt.Errorf("不支持的运算: %s", req.Op)
//	}
//
//	return &CalcResponse{
//		Expression: fmt.Sprintf("%.2f %s %.2f", req.A, req.Op, req.B),
//		Result:     math.Round(result*100) / 100,
//	}, nil
//}
//
//func main() {
//	ctx := context.Background()
//	// 创建weatherTool
//	weatherTool := utils.NewTool(
//		&schema.ToolInfo{
//			Name: "weather",
//			Desc: "获取指定城市的天气，输入城市名称，输出天气信息",
//			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
//				"city": {
//					Type:     schema.String,
//					Desc:     "要查询天气的城市名称，如北京、上海",
//					Required: true,
//				},
//			}),
//		},
//		getWeather,
//	)
//	calcTool := utils.NewTool(
//		&schema.ToolInfo{
//			Name: "calculator",
//			Desc: "对两个数字执行四则运算",
//			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
//				"a": {
//					Type:     schema.Number,
//					Desc:     "第一个数字",
//					Required: true,
//				},
//				"b": {
//					Type:     schema.Number,
//					Desc:     "第二个数字",
//					Required: true,
//				},
//				"op": {
//					Type:     schema.String,
//					Desc:     "运算类型",
//					Required: true,
//					Enum:     []string{"add", "subtract", "multiply", "divide"},
//				},
//			}),
//		},
//		calculate,
//	)
//	info, _ := weatherTool.Info(ctx)
//	fmt.Printf("工具名: %s\n", info.Name)
//	fmt.Printf("工具描述: %s\n", info.Desc)
//	args := `{"city": "北京"}`
//	result, err := weatherTool.InvokableRun(ctx, args)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("工具结果: %s\n", result)
//	var resp WeatherResponse
//	json.Unmarshal([]byte(result), &resp)
//	fmt.Println(resp)
//	fmt.Println()
//
//	// 模拟调用
//	result, err = calcTool.InvokableRun(ctx, `{"a": 12.5, "b": 3.7, "op": "multiply"}`)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result)
//}
