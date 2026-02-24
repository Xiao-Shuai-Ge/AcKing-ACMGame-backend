package logic

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"tgwp/global"
	"tgwp/model"
	"tgwp/repo"
	"tgwp/response"
	"tgwp/types"
)

type RoomExtraInfo struct {
	Submissions []types.RoomSubmissionRecord `json:"submissions"`
}

type SinglePlayerManager struct {
	mu      sync.Mutex
	workers map[int64]*singlePlayerWorker
}

type singlePlayerWorker struct {
	manager   *SinglePlayerManager
	room      model.SinglePlayerRoom
	problem   model.CodeforcesProblem
	processed map[int64]struct{}
	stopCh    chan struct{}
	penalty   int
}

var singlePlayerManagerOnce sync.Once
var singlePlayerManager *SinglePlayerManager

func GetSinglePlayerManager() *SinglePlayerManager {
	singlePlayerManagerOnce.Do(func() {
		singlePlayerManager = &SinglePlayerManager{
			workers: make(map[int64]*singlePlayerWorker),
		}
	})
	return singlePlayerManager
}

func (m *SinglePlayerManager) StartRoom(room model.SinglePlayerRoom, problem model.CodeforcesProblem) {
	m.mu.Lock()
	if _, ok := m.workers[room.ID]; ok {
		m.mu.Unlock()
		return
	}
	worker := &singlePlayerWorker{
		manager:   m,
		room:      room,
		problem:   problem,
		processed: make(map[int64]struct{}),
		stopCh:    make(chan struct{}),
		penalty:   room.Penalty,
	}

	if room.ExtraInfo != "" {
		var extraInfo RoomExtraInfo
		if err := json.Unmarshal([]byte(room.ExtraInfo), &extraInfo); err == nil {
			for _, s := range extraInfo.Submissions {
				worker.processed[s.SubmissionID] = struct{}{}
			}
		}
	}

	m.workers[room.ID] = worker
	m.mu.Unlock()
	go worker.run()
}

func (m *SinglePlayerManager) StopRoom(roomID int64) {
	m.mu.Lock()
	worker, ok := m.workers[roomID]
	if ok {
		close(worker.stopCh)
		delete(m.workers, roomID)
	}
	m.mu.Unlock()
}

func (w *singlePlayerWorker) run() {
	ticker := time.NewTicker(2 * time.Second)
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

func (w *singlePlayerWorker) tick() {
	submissions := GetCfQueue().GetUserSubmissions(w.room.UserID)
	if len(submissions) == 0 {
		return
	}
	for _, submission := range submissions {
		if submission.ProblemID != w.room.ProblemID {
			continue
		}
		if _, ok := w.processed[submission.SubmissionID]; ok {
			continue
		}
		if isPendingVerdict(submission.Verdict) {
			continue
		}
		if submission.Verdict == "OK" {
			w.processed[submission.SubmissionID] = struct{}{}
			w.saveSubmission(submission.SubmissionID, submission.Verdict)
			GetWsHub().SendToUser(w.room.UserID, types.WsResponse{
				Type:    "single_room_update",
				Code:    response.SUCCESS.Code,
				Message: response.SUCCESS.Msg,
				Data: map[string]interface{}{
					"room":         buildSingleRoomInfo(w.room, w.problem),
					"last_verdict": submission.Verdict,
				},
			})
			w.finish(true)
			return
		}
		w.processed[submission.SubmissionID] = struct{}{}
		w.saveSubmission(submission.SubmissionID, submission.Verdict)
		if isPenaltyVerdict(submission.Verdict) {
			w.penalty += 3
			_ = updateRoomPenalty(w.room.ID, w.penalty)
			w.room.Penalty = w.penalty
		}
		GetWsHub().SendToUser(w.room.UserID, types.WsResponse{
			Type:    "single_room_update",
			Code:    response.SUCCESS.Code,
			Message: response.SUCCESS.Msg,
			Data: map[string]interface{}{
				"room":         buildSingleRoomInfo(w.room, w.problem),
				"last_verdict": submission.Verdict,
			},
		})
	}
}

func isPendingVerdict(verdict string) bool {
	if verdict == "" {
		return true
	}
	switch verdict {
	case "TESTING", "SUBMITTED":
		return true
	default:
		return false
	}
}

func isPenaltyVerdict(verdict string) bool {
	switch verdict {
	case "RUNTIME_ERROR", "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED":
		return true
	default:
		return false
	}
}

func (w *singlePlayerWorker) finish(solved bool) {
	status := int8(1)
	if solved {
		status = 2
	}
	room, err := finishSingleRoom(context.Background(), w.room, w.problem.Difficulty, w.penalty, status)
	if err == nil {
		w.room = room
		GetWsHub().SendToUser(w.room.UserID, types.WsResponse{
			Type:    "single_room_finish",
			Code:    response.SUCCESS.Code,
			Message: response.SUCCESS.Msg,
			Data: map[string]interface{}{
				"room": buildSingleRoomInfo(w.room, w.problem),
			},
		})
	}
	w.manager.StopRoom(w.room.ID)
}

func updateRoomPenalty(roomID int64, penalty int) error {
	roomRepo := repo.NewSinglePlayerRoomRepo(global.DB)
	return roomRepo.UpdatePenalty(roomID, penalty)
}

func (w *singlePlayerWorker) saveSubmission(submissionID int64, verdict string) {
	var extraInfo RoomExtraInfo
	if w.room.ExtraInfo != "" {
		_ = json.Unmarshal([]byte(w.room.ExtraInfo), &extraInfo)
	}
	// check duplicate
	for _, s := range extraInfo.Submissions {
		if s.SubmissionID == submissionID {
			return
		}
	}
	extraInfo.Submissions = append(extraInfo.Submissions, types.RoomSubmissionRecord{
		SubmissionID: submissionID,
		Verdict:      verdict,
		SubmitTime:   time.Now().Unix(),
	})
	bytes, _ := json.Marshal(extraInfo)
	w.room.ExtraInfo = string(bytes)
	_ = repo.NewSinglePlayerRoomRepo(global.DB).UpdateExtraInfo(w.room.ID, w.room.ExtraInfo)
}
