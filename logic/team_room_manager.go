package logic

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"tgwp/global"
	"tgwp/log/zlog"
	"tgwp/model"
	"tgwp/repo"
	"tgwp/response"
	"tgwp/types"
)

const (
	teamRoomCheckInterval   = 2 * time.Second
	teamRoomPenaltyPerWrong = 20
)

type TeamRoomManager struct {
	mu      sync.Mutex
	workers map[int64]*teamRoomWorker
}

type teamRoomWorker struct {
	manager     *TeamRoomManager
	room        model.TeamRoom
	problems    map[string]teamRoomProblem
	statusList  []teamRoomProblemStatus
	submissions []teamRoomSubmissionRecord
	processed   map[int64]struct{}
	startTime   time.Time
	duration    time.Duration
	stopCh      chan struct{}
}

var teamRoomManagerOnce sync.Once
var teamRoomManager *TeamRoomManager

func GetTeamRoomManager() *TeamRoomManager {
	teamRoomManagerOnce.Do(func() {
		teamRoomManager = &TeamRoomManager{
			workers: make(map[int64]*teamRoomWorker),
		}
	})
	return teamRoomManager
}

func (m *TeamRoomManager) StartRoom(room model.TeamRoom) {
	m.mu.Lock()
	if _, ok := m.workers[room.ID]; ok {
		m.mu.Unlock()
		return
	}
	problems := parseTeamRoomProblems(room.ProblemList)
	problemMap := make(map[string]teamRoomProblem, len(problems))
	for _, p := range problems {
		problemMap[p.ProblemID] = p
	}
	statusList := parseTeamRoomProblemStatus(room.ProblemStatus)
	submissions := parseTeamRoomSubmissions(room.SubmissionRecords)
	processed := make(map[int64]struct{})
	for _, s := range submissions {
		processed[s.SubmissionID] = struct{}{}
	}
	worker := &teamRoomWorker{
		manager:     m,
		room:        room,
		problems:    problemMap,
		statusList:  statusList,
		submissions: submissions,
		processed:   processed,
		startTime:   room.CreatedAt,
		duration:    getTeamRoomDuration(room.Mode),
		stopCh:      make(chan struct{}),
	}
	if extra := parseTeamRoomExtra(room.ExtraInfo); extra.DurationSeconds > 0 {
		worker.duration = time.Duration(extra.DurationSeconds) * time.Second
	}
	m.workers[room.ID] = worker
	m.mu.Unlock()
	go worker.run()
}

func (m *TeamRoomManager) StopRoom(roomID int64) {
	m.mu.Lock()
	worker, ok := m.workers[roomID]
	if ok {
		close(worker.stopCh)
		delete(m.workers, roomID)
	}
	m.mu.Unlock()
}

func (w *teamRoomWorker) run() {
	ticker := time.NewTicker(teamRoomCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopCh:
			return
		}
	}
}

func (w *teamRoomWorker) tick() {
	if w.room.Status != 0 {
		return
	}
	if w.isTimeout() {
		w.finish(false)
		return
	}
	userIDs := GetWsHub().ActiveRoomUserIDs(w.room.ID)
	if len(userIDs) == 0 {
		return
	}
	for _, userID := range userIDs {
		submissions := GetCfQueue().GetUserSubmissions(userID)
		if len(submissions) == 0 {
			continue
		}
		for _, submission := range submissions {
			if submission.ProblemID == "" {
				continue
			}
			if _, ok := w.problems[submission.ProblemID]; !ok {
				continue
			}
			if _, ok := w.processed[submission.SubmissionID]; ok {
				continue
			}
			if isPendingVerdict(submission.Verdict) {
				continue
			}
			w.handleSubmission(userID, submission)
			if w.allSolved() {
				w.finish(true)
				return
			}
		}
	}
}

func (w *teamRoomWorker) handleSubmission(userID int64, submission CfSubmission) {
	w.processed[submission.SubmissionID] = struct{}{}
	submitTime := time.Now().Unix()
	w.submissions = append(w.submissions, teamRoomSubmissionRecord{
		SubmissionID: submission.SubmissionID,
		ProblemID:    submission.ProblemID,
		UserID:       userID,
		Verdict:      submission.Verdict,
		SubmitTime:   submitTime,
	})
	status := w.getProblemStatus(submission.ProblemID)
	changed := false
	if submission.Verdict == "OK" {
		if !status.Solved {
			status.Solved = true
			status.SolvedBy = userID
			status.SolvedAt = int64(time.Since(w.startTime).Seconds())
			changed = true
		}
	} else {
		if !status.Solved {
			status.Penalty += teamRoomPenaltyPerWrong
			changed = true
		}
	}
	if changed {
		w.setProblemStatus(status)
	}
	w.flushSubmissions()
	if changed {
		w.flushProblemStatus()
	}
	GetWsHub().SendToRoom(w.room.ID, types.WsResponse{
		Type:    "team_room_update",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":         buildTeamRoomInfo(w.room),
			"user_id":      userID,
			"problem_id":   submission.ProblemID,
			"last_verdict": submission.Verdict,
		},
	})
}

func (w *teamRoomWorker) getProblemStatus(problemID string) teamRoomProblemStatus {
	for _, item := range w.statusList {
		if item.ProblemID == problemID {
			return item
		}
	}
	return teamRoomProblemStatus{ProblemID: problemID}
}

func (w *teamRoomWorker) setProblemStatus(status teamRoomProblemStatus) {
	for i := range w.statusList {
		if w.statusList[i].ProblemID == status.ProblemID {
			w.statusList[i] = status
			return
		}
	}
	w.statusList = append(w.statusList, status)
}

func (w *teamRoomWorker) allSolved() bool {
	for _, item := range w.statusList {
		if !item.Solved {
			return false
		}
	}
	return len(w.statusList) > 0
}

func (w *teamRoomWorker) isTimeout() bool {
	if w.duration <= 0 {
		return false
	}
	return time.Since(w.startTime) >= w.duration
}

func (w *teamRoomWorker) flushSubmissions() {
	bytes, _ := json.Marshal(w.submissions)
	w.room.SubmissionRecords = string(bytes)
	_ = repo.NewTeamRoomRepo(global.DB).UpdateSubmissionRecords(w.room.ID, w.room.SubmissionRecords)
}

func (w *teamRoomWorker) flushProblemStatus() {
	bytes, _ := json.Marshal(w.statusList)
	w.room.ProblemStatus = string(bytes)
	_ = repo.NewTeamRoomRepo(global.DB).UpdateProblemStatus(w.room.ID, w.room.ProblemStatus)
}

func (w *teamRoomWorker) finish(allSolved bool) {
	if w.room.Status != 0 {
		return
	}
	score := int64(0)
	solvedCount := 0
	for _, item := range w.statusList {
		if !item.Solved {
			continue
		}
		score += item.SolvedAt + int64(item.Penalty*60)
		solvedCount++
	}
	extra := parseTeamRoomExtra(w.room.ExtraInfo)
	extra.Score = score
	if extra.DurationSeconds == 0 {
		extra.DurationSeconds = int64(w.duration.Seconds())
	}
	extraBytes, _ := json.Marshal(extra)
	w.room.ExtraInfo = string(extraBytes)
	w.room.Status = 1
	w.room.EndTime = time.Now().Unix()
	w.flushSubmissions()
	w.flushProblemStatus()
	_ = repo.NewTeamRoomRepo(global.DB).UpdateExtraInfo(w.room.ID, w.room.ExtraInfo)
	_ = repo.NewTeamRoomRepo(global.DB).UpdateStatus(w.room.ID, w.room.Status, w.room.EndTime)
	GetWsHub().SendToRoom(w.room.ID, types.WsResponse{
		Type:    "team_room_finish",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":         buildTeamRoomInfo(w.room),
			"all_solved":   allSolved,
			"solved_count": solvedCount,
		},
	})
	w.manager.StopRoom(w.room.ID)
}

func StartAllActiveTeamRooms() error {
	roomRepo := repo.NewTeamRoomRepo(global.DB)
	rooms, err := roomRepo.ListActive()
	if err != nil {
		zlog.Errorf("初始化团队房间失败：%v", err)
		return err
	}
	if len(rooms) == 0 {
		return nil
	}
	for _, room := range rooms {
		GetTeamRoomManager().StartRoom(room)
	}
	return nil
}

func FinishAllActiveTeamRooms(ctx context.Context) {
	roomRepo := repo.NewTeamRoomRepo(global.DB)
	rooms, err := roomRepo.ListActive()
	if err != nil {
		zlog.CtxWarnf(ctx, "团队房间结算失败：%v", err)
		return
	}
	manager := GetTeamRoomManager()
	for _, room := range rooms {
		manager.mu.Lock()
		worker := manager.workers[room.ID]
		manager.mu.Unlock()
		if worker != nil {
			worker.finish(false)
			continue
		}
		room.Status = 1
		room.EndTime = time.Now().Unix()
		_ = roomRepo.UpdateStatus(room.ID, room.Status, room.EndTime)
	}
}
