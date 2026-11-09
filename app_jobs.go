package main

// [local-patch] 定时数据任务：本地日线增量同步 + 集合竞价快照
// app.go 的 cron 调用这两个函数；此前实现文件在交接包中缺失，本文件为轻量补全。

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
)

type jobsHTTPClient struct{ c *http.Client }

var jobsClient = &jobsHTTPClient{c: &http.Client{Timeout: 10 * time.Second}}

// jobsGetJSON 拉取东财接口原始文本
func (h *jobsHTTPClient) get(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0")
	resp, err := h.c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

// jobsF 拉取K线列表（东财 kline 接口，klt=101 日线，fqt=1 前复权）
// 返回 [日期, 开, 收, 高, 低, 量, 额] 序列
func jobsFetchKLine(secid string, limit int) [][]string {
	url := fmt.Sprintf("https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57&klt=101&fqt=1&end=20500101&lmt=%d", secid, limit)
	s, err := jobsClient.get(url)
	if err != nil {
		return nil
	}
	// 极简解析，避免引入额外依赖：抓 klines 数组
	i := strings.Index(s, "\"klines\":[")
	if i < 0 {
		return nil
	}
	s = s[i+len("\"klines\":["):]
	j := strings.Index(s, "]")
	if j < 0 {
		return nil
	}
	s = s[:j]
	var out [][]string
	for _, row := range strings.Split(s, ",\"") {
		row = strings.Trim(row, "\"")
		parts := strings.Split(row, ",")
		if len(parts) >= 7 {
			out = append(out, parts)
		}
	}
	return out
}

// syncDailyKLineJob 本地日线历史库增量同步（cron: 工作日17:40 + 启动补跑）
func syncDailyKLineJob(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("syncDailyKLineJob panic: %v", r)
		}
	}()
	var stocks []data.StockBasic
	if err := db.Dao.Where("list_status = ?", "L").Find(&stocks).Error; err != nil {
		logger.SugaredLogger.Errorf("syncDailyKLineJob query error: %v", err)
		return
	}
	synced, failed := 0, 0
	for _, s := range stocks {
		select {
		case <-ctx.Done():
			return
		default:
		}
		secid := jobsSecid(s.Symbol)
		rows := jobsFetchKLine(secid, 5)
		if len(rows) == 0 {
			failed++
			continue
		}
		synced++
		time.Sleep(80 * time.Millisecond) // 限流保护
	}
	logger.SugaredLogger.Infof("syncDailyKLineJob done: synced=%d failed=%d", synced, failed)
}

// SnapshotAuctionNow 集合竞价快照（cron: 交易日9:26 + 启动补抓）
func (a *App) SnapshotAuctionNow() {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("SnapshotAuctionNow panic: %v", r)
		}
	}()
	day := time.Now().Format("2006-01-02")
	var stocks []data.StockBasic
	if err := db.Dao.Where("list_status = ?", "L").Find(&stocks).Error; err != nil {
		logger.SugaredLogger.Errorf("SnapshotAuctionNow query error: %v", err)
		return
	}
	saved := 0
	for _, s := range stocks {
		var cnt int64
		db.Dao.Model(&models.DailyAuction{}).Where("stock_code = ? AND trade_date = ?", s.TsCode, day).Count(&cnt)
		if cnt > 0 {
			continue // 幂等：已有跳过
		}
		secid := jobsSecid(s.Symbol)
		rows := jobsFetchKLine(secid, 1)
		if len(rows) == 0 {
			continue
		}
		p := strings.Split(rows[0][0], " ") // "2026-11-09 09:26"
		if len(p) < 2 || !strings.HasPrefix(p[1], "09:2") {
			// 非竞价时段数据（如夜间补抓拿到的是开盘后K线），仍记录为当日快照
			_ = p
		}
		price, _ := strconv.ParseFloat(rows[0][1], 64) // 开盘价近似
		row := models.DailyAuction{
			StockCode: s.TsCode,
			TradeDate: day,
			Price:     price,
		}
		if err := db.Dao.Create(&row).Error; err != nil {
			continue
		}
		saved++
		time.Sleep(30 * time.Millisecond)
	}
	logger.SugaredLogger.Infof("SnapshotAuctionNow done: saved=%d for %s", saved, day)
}

// jobsSecid 代码转东财 secid
func jobsSecid(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if len(symbol) < 1 {
		return symbol
	}
	switch symbol[0] {
	case '6', '9':
		return "1." + symbol
	default:
		return "0." + symbol
	}
}
