package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/server"
	ai "hk_stock_assistant/backend/ai_service/kitex_gen/ai/aiservice"
	"hk_stock_assistant/backend/ai_service/biz/predictor"
	"hk_stock_assistant/backend/stock_service/kitex_gen/stock/stockservice"
)

func init() {
	// 加载当前工作目录下的 .env（仅对未设置的环境变量生效）
	dir, _ := os.Getwd()
	if dir == "" {
		dir = "."
	}
	envPath := filepath.Join(dir, ".env")
	f, err := os.Open(envPath)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" && val != "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	stockClient, err := stockservice.NewClient("stock_service", client.WithHostPorts("127.0.0.1:8888"))
	if err != nil {
		log.Fatalf("init stock client: %v", err)
	}
	p := predictor.New(stockClient)
	go RunStreamServer(p)
	addr, _ := net.ResolveTCPAddr("tcp", ":8889")
	svr := ai.NewServer(NewAIServiceImpl(stockClient, p), server.WithServiceAddr(addr))
	if err := svr.Run(); err != nil {
		log.Fatal(err)
	}
}
