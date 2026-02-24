package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"tgwp/global"
	"tgwp/log/zlog"
	"tgwp/repo"
)

const (
	cfScanInterval    = 2 * time.Second
	cfRequestInterval = 200 * time.Millisecond
	cfMaxSubmissions  = 30
	cfRequestTimeout  = 10 * time.Second
)

type CfSubmission struct {
	SubmissionID int64  `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	Verdict      string `json:"verdict"`
}

type CfQueue struct {
	mu          sync.RWMutex
	submissions map[int64][]CfSubmission
	handles     map[int64]string
	queue       []int64
	queued      map[int64]struct{}
	scanTicker  *time.Ticker
	reqTicker   *time.Ticker
	stopCh      chan struct{}
	running     int32
}

var cfQueueOnce sync.Once
var cfQueue *CfQueue

func GetCfQueue() *CfQueue {
	cfQueueOnce.Do(func() {
		cfQueue = &CfQueue{
			submissions: make(map[int64][]CfSubmission),
			handles:     make(map[int64]string),
			queue:       make([]int64, 0),
			queued:      make(map[int64]struct{}),
		}
	})
	return cfQueue
}

func StartCfQueue() {
	GetCfQueue().Start(cfScanInterval, cfRequestInterval)
}

func StopCfQueue() {
	GetCfQueue().Stop()
}

func (q *CfQueue) Start(scanInterval, requestInterval time.Duration) {
	if !atomic.CompareAndSwapInt32(&q.running, 0, 1) {
		return
	}
	q.scanTicker = time.NewTicker(scanInterval)
	q.reqTicker = time.NewTicker(requestInterval)
	q.stopCh = make(chan struct{})
	go q.scanLoop()
	go q.requestLoop()
}

func (q *CfQueue) Stop() {
	if !atomic.CompareAndSwapInt32(&q.running, 1, 0) {
		return
	}
	if q.scanTicker != nil {
		q.scanTicker.Stop()
	}
	if q.reqTicker != nil {
		q.reqTicker.Stop()
	}
	if q.stopCh != nil {
		close(q.stopCh)
	}
}

func (q *CfQueue) scanLoop() {
	for {
		select {
		case <-q.scanTicker.C:
			userIDs := GetWsHub().ActiveUserIDs()
			for _, userID := range userIDs {
				q.enqueue(userID)
			}
		case <-q.stopCh:
			return
		}
	}
}

func (q *CfQueue) requestLoop() {
	for {
		select {
		case <-q.reqTicker.C:
			userID, ok := q.pop()
			if !ok {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), cfRequestTimeout)
			handle, ok := q.getHandle(ctx, userID)
			if !ok {
				cancel()
				continue
			}
			submissions, err := q.fetchSubmissions(ctx, handle)
			if err != nil {
				cancel()
				zlog.CtxWarnf(ctx, "Codeforces请求失败:%v", err)
				continue
			}
			cancel()
			q.setSubmissions(userID, submissions)
		case <-q.stopCh:
			return
		}
	}
}

func (q *CfQueue) enqueue(userID int64) {
	if userID == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.queued[userID]; ok {
		return
	}
	q.queue = append(q.queue, userID)
	q.queued[userID] = struct{}{}
}

func (q *CfQueue) pop() (int64, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.queue) == 0 {
		return 0, false
	}
	userID := q.queue[0]
	q.queue = q.queue[1:]
	delete(q.queued, userID)
	return userID, true
}

func (q *CfQueue) getHandle(ctx context.Context, userID int64) (string, bool) {
	q.mu.RLock()
	handle, ok := q.handles[userID]
	q.mu.RUnlock()
	if ok && handle != "" {
		return handle, true
	}
	userRepo := repo.NewUserRepo(global.DB)
	user, err := userRepo.GetByID(userID)
	if err != nil {
		zlog.CtxWarnf(ctx, "获取用户失败:%v", err)
		return "", false
	}
	if user.Username == "" {
		return "", false
	}
	q.mu.Lock()
	q.handles[userID] = user.Username
	q.mu.Unlock()
	return user.Username, true
}

func (q *CfQueue) SetUserHandle(userID int64, handle string) {
	if userID == 0 || handle == "" {
		return
	}
	q.mu.Lock()
	q.handles[userID] = handle
	q.mu.Unlock()
}

func (q *CfQueue) setSubmissions(userID int64, submissions []CfSubmission) {
	if len(submissions) > cfMaxSubmissions {
		submissions = submissions[:cfMaxSubmissions]
	}
	items := make([]CfSubmission, len(submissions))
	copy(items, submissions)
	q.mu.Lock()
	q.submissions[userID] = items
	q.mu.Unlock()
}

func (q *CfQueue) GetUserSubmissions(userID int64) []CfSubmission {
	q.mu.RLock()
	defer q.mu.RUnlock()
	items := q.submissions[userID]
	if len(items) == 0 {
		return nil
	}
	result := make([]CfSubmission, len(items))
	copy(result, items)
	return result
}

type cfStatusResponse struct {
	Status  string         `json:"status"`
	Result  []cfSubmission `json:"result"`
	Comment string         `json:"comment"`
}

type cfSubmission struct {
	ID      int64     `json:"id"`
	Verdict string    `json:"verdict"`
	Problem cfProblem `json:"problem"`
}

type cfProblem struct {
	ContestID int    `json:"contestId"`
	Index     string `json:"index"`
}

func (q *CfQueue) fetchSubmissions(ctx context.Context, handle string) ([]CfSubmission, error) {
	client := &http.Client{
		Timeout: cfRequestTimeout,
	}
	url := fmt.Sprintf("https://codeforces.com/api/user.status?handle=%s&from=1&count=%d", handle, cfMaxSubmissions)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP状态码异常:%d", resp.StatusCode)
	}
	var data cfStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Status != "OK" {
		if data.Comment != "" {
			return nil, fmt.Errorf("API返回失败:%s", data.Comment)
		}
		return nil, fmt.Errorf("API返回失败")
	}
	items := make([]CfSubmission, 0, len(data.Result))
	for _, item := range data.Result {
		problemID := ""
		if item.Problem.ContestID > 0 && item.Problem.Index != "" {
			problemID = fmt.Sprintf("%d%s", item.Problem.ContestID, item.Problem.Index)
		}
		items = append(items, CfSubmission{
			SubmissionID: item.ID,
			ProblemID:    problemID,
			Verdict:      item.Verdict,
		})
		if len(items) >= cfMaxSubmissions {
			break
		}
	}
	return items, nil
}
