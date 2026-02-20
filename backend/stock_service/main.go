package main

import (
	"log"
	"net"

	"github.com/cloudwego/kitex/server"
	stock "hk_stock_assistant/backend/stock_service/kitex_gen/stock/stockservice"
)

func main() {
	addr, _ := net.ResolveTCPAddr("tcp", ":8888")
	svr := stock.NewServer(NewStockServiceImpl(), server.WithServiceAddr(addr))
	if err := svr.Run(); err != nil {
		log.Fatal(err)
	}
}
