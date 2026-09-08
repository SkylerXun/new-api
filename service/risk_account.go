package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type riskAccountBackfillHandler struct{}

type RiskAccountBackfillPayload struct {
	RunID string `json:"run_id"`
}

type RiskAccountBackfillState struct {
	RunID          string `json:"run_id"`
	LastUserID     int    `json:"last_user_id"`
	ProcessedUsers int64  `json:"processed_users"`
	TotalUsers     int64  `json:"total_users"`
	Progress       int    `json:"progress"`
}

func (riskAccountBackfillHandler) Type() string { return model.SystemTaskTypeRiskBackfill }

func (riskAccountBackfillHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := RiskAccountBackfillPayload{}
	if err := task.DecodePayload(&payload); err != nil || payload.RunID == "" {
		if err == nil {
			err = errors.New("risk backfill run id is required")
		}
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, err.Error())
		return
	}
	run, err := model.GetRiskAccountBackfillRun(payload.RunID)
	if err != nil {
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, err.Error())
		return
	}
	if run.Status == model.RiskBackfillStatusPauseRequested {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusPaused, "")
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, map[string]any{"paused": true, "run_id": run.RunID}, "")
		return
	}
	if err := model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusRunning, ""); err != nil {
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, err.Error())
		return
	}
	preview, err := model.PreviewRiskAccountBackfill(ctx, run, func(lastUserID int, processed int64) error {
		if err := model.UpdateRiskAccountBackfillProgress(run.RunID, lastUserID, processed); err != nil {
			return err
		}
		progress := 0
		if run.TotalUsers > 0 {
			progress = int(processed * 100 / run.TotalUsers)
			if progress > 100 {
				progress = 100
			}
		}
		return model.UpdateSystemTaskState(task.TaskID, runnerID, RiskAccountBackfillState{
			RunID: run.RunID, LastUserID: lastUserID, ProcessedUsers: processed,
			TotalUsers: run.TotalUsers, Progress: progress,
		})
	})
	if errors.Is(err, model.ErrRiskBackfillPaused) {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusPaused, "")
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, map[string]any{"paused": true, "run_id": run.RunID}, "")
		return
	}
	if err != nil {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusFailed, err.Error())
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, err.Error())
		return
	}
	if err := model.CompleteRiskAccountBackfillRun(run.RunID, preview); err != nil {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusFailed, err.Error())
		_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, err.Error())
		return
	}
	result := map[string]any{
		"run_id": run.RunID, "candidate_groups": preview.CandidateGroups,
		"candidate_accounts": preview.CandidateAccounts,
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, result, ""); err != nil {
		common.SysError(fmt.Sprintf("failed to finish risk account backfill task: %v", err))
	}
}

func init() { RegisterSystemTaskHandler(riskAccountBackfillHandler{}) }

func activeRiskAccountBackfill() (*model.RiskAccountBackfillRun, *model.SystemTask, error) {
	active, err := model.GetActiveSystemTask(model.SystemTaskTypeRiskBackfill)
	if err != nil || active == nil {
		return nil, active, err
	}
	payload := RiskAccountBackfillPayload{}
	if err := active.DecodePayload(&payload); err != nil {
		return nil, nil, err
	}
	run, err := model.GetRiskAccountBackfillRun(payload.RunID)
	return run, active, err
}

func StartRiskAccountBackfill(options model.RiskBackfillOptions, createdBy int) (*model.RiskAccountBackfillRun, *model.SystemTask, bool, error) {
	if activeRun, activeTask, err := activeRiskAccountBackfill(); err != nil {
		return nil, nil, false, err
	} else if activeTask != nil {
		return activeRun, activeTask, false, nil
	}
	run, err := model.CreateRiskAccountBackfillRun(options, createdBy)
	if err != nil {
		return nil, nil, false, err
	}
	task, created, err := EnqueueSystemTask(model.SystemTaskTypeRiskBackfill, RiskAccountBackfillPayload{RunID: run.RunID})
	if err != nil {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusFailed, err.Error())
		return nil, nil, false, err
	}
	if !created {
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusFailed, "another risk scan is already active")
		activeRun, activeTask, activeErr := activeRiskAccountBackfill()
		return activeRun, activeTask, false, activeErr
	}
	if err := model.SetRiskAccountBackfillTask(run.RunID, task.TaskID); err != nil {
		return nil, nil, false, err
	}
	run.TaskID = task.TaskID
	return run, task, true, nil
}

func ResumeRiskAccountBackfill(runID string) (*model.RiskAccountBackfillRun, *model.SystemTask, error) {
	if _, activeTask, err := activeRiskAccountBackfill(); err != nil {
		return nil, nil, err
	} else if activeTask != nil {
		return nil, nil, errors.New("another risk scan is already active")
	}
	run, err := model.PrepareRiskAccountBackfillResume(runID)
	if err != nil {
		return nil, nil, err
	}
	task, created, err := EnqueueSystemTask(model.SystemTaskTypeRiskBackfill, RiskAccountBackfillPayload{RunID: run.RunID})
	if err != nil || !created {
		if err == nil {
			err = errors.New("another risk scan is already active")
		}
		_ = model.MarkRiskAccountBackfillStatus(run.RunID, model.RiskBackfillStatusPaused, err.Error())
		return nil, nil, err
	}
	if err := model.SetRiskAccountBackfillTask(run.RunID, task.TaskID); err != nil {
		return nil, nil, err
	}
	run.TaskID = task.TaskID
	return run, task, nil
}
