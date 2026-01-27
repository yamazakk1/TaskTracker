package worker

import (
	"context"
	"taskTracker/internal/service"
	"taskTracker/internal/models/task"
	"taskTracker/internal/logger"
	"time"
	"fmt"
	"go.uber.org/zap"
)

type OverdueWorker struct{
	repo service.TaskRepository
	interval time.Duration
	batchSize int
}


func NewOverdueWorker(repo service.TaskRepository, interval *time.Duration, batchSize *int) *OverdueWorker {
	var intervalToSet time.Duration 
	if interval == nil{
		intervalToSet = 5 * time.Minute 
	} else{
		intervalToSet = *interval
	}

	var batchToSet int 
	if batchSize == nil{
		batchToSet = 100
	}else{
		batchToSet = *batchSize
	}
    return &OverdueWorker{
        repo:      repo,
        interval:  intervalToSet,  
        batchSize: batchToSet,              
    }
}

func (w *OverdueWorker) Start(ctx context.Context){
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for{
		select{
		case <-ticker.C:
			logger.Info("Worker: Фоновая проверка задач на просроченность",zap.Time("started_at", time.Now()))
			w.Check(ctx)
		case <-ctx.Done():
			logger.Info("Worker: Фоновая проверка останавливается")
			return
		}
	}
}

func (w *OverdueWorker) Check(ctx context.Context){
	start := time.Now()

	tasks, err := w.getAllActiveTasks(ctx)
	if err != nil{
		logger.Warn("Worker: ошибка получения задач", zap.Error(err))
		return
	}

	overdueCount := 0
    now := time.Now()

	for _, t := range tasks{
		if t.Status == task.StatusDone || t.Status == task.StatusOverdue{
			continue
		}

		if t.DueTime.Before(now){
			if err := w.MarkAsOverdue(ctx, t); err != nil{
				logger.Warn("Worker: Ошибка обновления задачи", zap.Error(err))
				continue
			}
			overdueCount++
		}

		if overdueCount > w.batchSize{
			break
		}
	}
	duration := time.Since(start)
	logger.Info(
		"Worker: Завершение проверки задач", 
		zap.Duration("ms", duration), 
		zap.Int("checked", len(tasks)),
		zap.Int("overdue", overdueCount),
	)

}


func (w *OverdueWorker) getAllActiveTasks(ctx context.Context) ([]*task.Task, error) {

    tasks, err := w.repo.GetTasksDueBefore(ctx, time.Now(), w.batchSize)
    if err != nil {
        return nil, fmt.Errorf("получение активных задач: %w", err)
    }
    return tasks, nil
}

func (w *OverdueWorker) MarkAsOverdue(ctx context.Context, t *task.Task) (error){
	t.Status = task.StatusOverdue

	err := w.repo.Update(ctx, t)
	if err != nil{
		return fmt.Errorf("обновление статуса: %w", err)
	}
	return nil
}