package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// 东方财富 港股 列表接口：涨跌幅、资金流向（f2 港元、f3 已为百分比%，无需换算）
const (
	stockListURL = "https://push2.eastmoney.com/api/qt/clist/get"
	stockListUT  = "fa5fd1943c7b386f172d6893dbfba10b"
	stockFields  = "f12,f14,f2,f3,f62,f184,f66,f72,f78,f84" // +超大单/大单/中单/小单净流入(元)，用于汇总今日主力/散户
	stockFS      = "m:116+t:3"                             // 港股
)

var stockListHTTP = &http.Client{Timeout: 12 * time.Second}

type stockListItem struct {
	F12  string  `json:"f12"`  // 股票代码
	F14  string  `json:"f14"`  // 股票名称
	F2   float64 `json:"f2"`   // 最新价
	F3   float64 `json:"f3"`   // 涨跌幅%
	F62  float64 `json:"f62"`  // 主力净流入(元)
	F184 float64 `json:"f184"` // 主力净占比%
	F66  float64 `json:"f66"`  // 超大单净流入
	F72  float64 `json:"f72"`  // 大单净流入
	F78  float64 `json:"f78"`  // 中单净流入
	F84  float64 `json:"f84"`  // 小单净流入
}

type stockListResp struct {
	Data struct {
		Total int              `json:"total"`
		Diff  []stockListItem `json:"diff"`
	} `json:"data"`
}

// GetSectors 获取股票列表（热点股票、涨跌幅排行、资金流向），返回结构兼容前端
func GetSectors(ctx context.Context, c *app.RequestContext) {
	url := fmt.Sprintf("%s?pn=1&pz=200&po=1&np=1&fltt=2&invt=2&fid=f3&fs=%s&fields=%s&ut=%s",
		stockListURL, stockFS, stockFields, stockListUT)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := stockListHTTP.Do(req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var r stockListResp
	if err := json.Unmarshal(body, &r); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"error": "parse response"})
		return
	}
	diff := r.Data.Diff
	if len(diff) == 0 {
		c.JSON(consts.StatusOK, map[string]interface{}{
			"hot":         []map[string]interface{}{},
			"by_change":   []map[string]interface{}{},
			"by_capital":  []map[string]interface{}{},
		})
		return
	}

	// 热点股票：按涨跌幅取前 10
	byChange := make([]stockListItem, len(diff))
	copy(byChange, diff)
	sort.Slice(byChange, func(i, j int) bool { return byChange[i].F3 > byChange[j].F3 })
	hot := byChange
	if len(hot) > 10 {
		hot = hot[:10]
	}

	// 资金流向排行：按主力净流入 f62 降序
	byCapital := make([]stockListItem, len(diff))
	copy(byCapital, diff)
	sort.Slice(byCapital, func(i, j int) bool { return byCapital[i].F62 > byCapital[j].F62 })
	topCapital := byCapital
	if len(topCapital) > 20 {
		topCapital = topCapital[:20]
	}

	// 港股 clist：f2 为港元，f3 已为百分比；f62 主力净流入，f78+f84 散户净流入
	toMap := func(s stockListItem) map[string]interface{} {
		value := s.F2
		changePct := s.F3
		return map[string]interface{}{
			"code":              s.F12,
			"name":              s.F14,
			"value":             value,
			"change_percent":    changePct,
			"main_net_inflow":   s.F62,
			"main_net_ratio":    s.F184,
			"retail_net_inflow": s.F78 + s.F84,
		}
	}
	hotList := make([]map[string]interface{}, 0, len(hot))
	for _, s := range hot {
		hotList = append(hotList, toMap(s))
	}
	changeList := make([]map[string]interface{}, 0, len(byChange))
	for _, s := range byChange {
		changeList = append(changeList, toMap(s))
	}
	capitalList := make([]map[string]interface{}, 0, len(topCapital))
	for _, s := range topCapital {
		capitalList = append(capitalList, toMap(s))
	}
	if len(topCapital) > 0 {
		first := topCapital[0]
		log.Printf("[sectors] capital sample: %s f62=%.0f f78=%.0f f84=%.0f retail_net=%.0f",
			first.F14, first.F62, first.F78, first.F84, first.F78+first.F84)
	}
	c.JSON(consts.StatusOK, map[string]interface{}{
		"hot":        hotList,
		"by_change":  changeList,
		"by_capital": capitalList,
	})
}
