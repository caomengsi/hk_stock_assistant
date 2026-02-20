package sina_hk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hk_stock_assistant/backend/stock_service/kitex_gen/stock"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Client handles interaction with Sina Finance HK stock API
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Sina HK API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// NormalizeHKCode ensures code is hk + 5 digits (e.g. 700 -> hk00700)
func NormalizeHKCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ToLower(code)
	if strings.HasPrefix(code, "hk") {
		if len(code) == 7 {
			return code
		}
		if len(code) == 6 {
			return "hk0" + code[2:]
		}
		return code
	}
	// digits only: pad to 5
	if len(code) <= 5 {
		return "hk" + strings.Repeat("0", 5-len(code)) + code
	}
	return "hk" + code
}

// GetStockInfo fetches real-time HK stock information
// code format: hk00700, hk09988, or 700, 9988
func (c *Client) GetStockInfo(ctx context.Context, code string) (*stock.StockInfo, error) {
	code = NormalizeHKCode(code)
	url := fmt.Sprintf("http://hq.sinajs.cn/list=%s", code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var content string
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Body, _, err := transform.Bytes(decoder, rawBody)
	if err != nil {
		content = string(rawBody)
	} else {
		content = string(utf8Body)
	}

	// Sina HK format: var hq_str_hk00700="腾讯控股, 350.200, 348.000, ...";
	if !strings.Contains(content, "=\"") {
		return nil, fmt.Errorf("invalid stock code or empty response: %s", code)
	}

	parts := strings.Split(content, "=\"")
	if len(parts) < 2 {
		return nil, fmt.Errorf("parse error")
	}
	dataStr := strings.TrimSuffix(parts[1], "\";")
	dataStr = strings.TrimSuffix(dataStr, "\"")
	if dataStr == "" {
		return nil, fmt.Errorf("empty data")
	}

	fields := strings.Split(dataStr, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	// Sina HK: 0=name_en, 1=name_cn, 2=open, 3=prevClose, 4=high, 5=low, 6=current, 7=change, 8=change%, 9=bid, 10=ask, 11=amount(HKD), 12=volume
	if len(fields) < 9 {
		return nil, fmt.Errorf("unexpected data format: %d fields", len(fields))
	}

	name := fields[1]
	if name == "" {
		name = fields[0]
	}
	currentPrice, _ := strconv.ParseFloat(fields[6], 64)
	changePercent, _ := strconv.ParseFloat(fields[8], 64)
	volume, _ := strconv.ParseInt(fields[12], 10, 64)
	if len(fields) <= 12 {
		volume = 0
	}
	if len(fields) > 17 {
		date := fields[16]
		timeStr := fields[17]
		return &stock.StockInfo{
			Code:          code,
			Name:          name,
			CurrentPrice:  currentPrice,
			ChangePercent: changePercent,
			Volume:        volume,
			Timestamp:     fmt.Sprintf("%s %s", date, timeStr),
		}, nil
	}
	return &stock.StockInfo{
		Code:          code,
		Name:          name,
		CurrentPrice:  currentPrice,
		ChangePercent: changePercent,
		Volume:        volume,
		Timestamp:     "",
	}, nil
}

// GetIndexInfo fetches index data (e.g. int_hangseng for Hang Seng Index)
func (c *Client) GetIndexInfo(ctx context.Context, listCode string) (name string, value, changePercent float64, err error) {
	url := fmt.Sprintf("http://hq.sinajs.cn/list=%s", listCode)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, err
	}

	var content string
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Body, _, decErr := transform.Bytes(decoder, rawBody)
	if decErr != nil {
		content = string(rawBody)
	} else {
		content = string(utf8Body)
	}

	if !strings.Contains(content, "=\"") {
		return "", 0, 0, fmt.Errorf("invalid response")
	}
	parts := strings.Split(content, "=\"")
	dataStr := strings.TrimSuffix(strings.TrimSuffix(parts[1], "\";"), "\"")
	fields := strings.Split(dataStr, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) < 3 {
		return "", 0, 0, fmt.Errorf("not enough fields")
	}
	name = fields[0]
	value, _ = strconv.ParseFloat(fields[1], 64)
	changePercent, _ = strconv.ParseFloat(fields[2], 64)
	return name, value, changePercent, nil
}
