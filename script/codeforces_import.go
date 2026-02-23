package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"tgwp/global"
	"tgwp/initalize"
	"tgwp/log/zlog"
	"tgwp/model"
	"time"

	"gorm.io/gorm/clause"
)

// go run .\script\codeforces_import.go -c .\config.yaml

type cfResponse struct {
	Status string `json:"status"`
	Result struct {
		Problems []cfProblem `json:"problems"`
	} `json:"result"`
	Comment string `json:"comment"`
}

type cfProblem struct {
	ContestID int    `json:"contestId"`
	Index     string `json:"index"`
	Rating    int    `json:"rating"`
}

func main() {
	initalize.Init()
	defer initalize.Eve()

	ctx := context.Background()
	problems, err := fetchProblems(ctx)
	if err != nil {
		zlog.CtxErrorf(ctx, "获取 Codeforces 题库失败：%v", err)
		return
	}

	items := make([]model.CodeforcesProblem, 0, len(problems))
	for _, p := range problems {
		if p.ContestID == 0 || p.Index == "" {
			continue
		}
		url := fmt.Sprintf("https://codeforces.com/problemset/problem/%d/%s", p.ContestID, p.Index)
		items = append(items, model.CodeforcesProblem{
			ID:         fmt.Sprintf("%d%s", p.ContestID, p.Index),
			Url:        url,
			Difficulty: p.Rating,
		})
	}

	if len(items) == 0 {
		zlog.CtxInfof(ctx, "未解析到可写入的题目")
		return
	}

	batchSize := 500
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		err = global.DB.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(items[i:end]).Error
		if err != nil {
			zlog.CtxErrorf(ctx, "写入题目失败：%v", err)
			return
		}
	}

	zlog.CtxInfof(ctx, "写入完成，总数：%d", len(items))
}

func fetchProblems(ctx context.Context) ([]cfProblem, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://codeforces.com/api/problemset.problems", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP状态码异常：%d", resp.StatusCode)
	}

	var data cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Status != "OK" {
		if data.Comment != "" {
			return nil, fmt.Errorf("API返回失败：%s", data.Comment)
		}
		return nil, fmt.Errorf("API返回失败")
	}
	return data.Result.Problems, nil
}
