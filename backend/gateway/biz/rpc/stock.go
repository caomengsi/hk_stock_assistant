package rpc

import (
	"sync"

	"github.com/cloudwego/kitex/client"
	"hk_stock_assistant/backend/stock_service/kitex_gen/stock/stockservice"
)

var (
	StockClient stockservice.Client
	stockOnce   sync.Once
)

func InitStock() {
	stockOnce.Do(func() {
		var err error
		StockClient, err = stockservice.NewClient("stock_service", client.WithHostPorts("127.0.0.1:8888"))
		if err != nil {
			panic(err)
		}
	})
}
