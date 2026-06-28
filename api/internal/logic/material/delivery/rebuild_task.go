package delivery

import (
	"api/internal/svc"
	"api/model"
	taskCode "api/pkg/code"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const rebuildTaskTimeout = 30 * time.Minute

type rebuildTaskMessage struct {
	taskId primitive.ObjectID
	svcCtx *svc.ServiceContext
}

var (
	rebuildTaskQueue    = make(chan rebuildTaskMessage, 1)
	rebuildWorkerOnce   sync.Once
	rebuildTaskMu       sync.Mutex
	activeRebuildTaskId string
)

// EnqueueRebuildTask creates a rebuild task and sends it to the background worker.
func EnqueueRebuildTask(ctx context.Context, svcCtx *svc.ServiceContext, creatorId, creatorName string) (model.MaterialDeliveryRebuildTask, bool, error) {
	startRebuildWorker()

	rebuildTaskMu.Lock()
	defer rebuildTaskMu.Unlock()

	if activeRebuildTaskId != "" {
		task, err := findRebuildTaskById(ctx, svcCtx, activeRebuildTaskId)
		if err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				return model.MaterialDeliveryRebuildTask{}, false, err
			}
			activeRebuildTaskId = ""
		} else {
			return task, false, nil
		}
	}

	if err := markInterruptedRebuildTasks(ctx, svcCtx); err != nil {
		return model.MaterialDeliveryRebuildTask{}, false, err
	}

	now := time.Now().Unix()
	task := model.MaterialDeliveryRebuildTask{
		Id:          primitive.NewObjectID(),
		Status:      taskCode.MaterialDeliveryRebuildTaskStatusQueued,
		Message:     "任务已进入后台队列",
		CreatorId:   creatorId,
		CreatorName: creatorName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := svcCtx.MaterialDeliveryRebuildTaskModel.InsertOne(ctx, &task); err != nil {
		return model.MaterialDeliveryRebuildTask{}, false, fmt.Errorf("创建客户新增物料重建任务失败:%w", err)
	}

	activeRebuildTaskId = task.Id.Hex()
	select {
	case rebuildTaskQueue <- rebuildTaskMessage{taskId: task.Id, svcCtx: svcCtx}:
		return task, true, nil
	default:
		activeRebuildTaskId = ""
		_ = updateRebuildTaskStatus(svcCtx, task.Id, taskCode.MaterialDeliveryRebuildTaskStatusFailed, 0, 0, "任务队列已满", "任务队列已满")
		task.Status = taskCode.MaterialDeliveryRebuildTaskStatusFailed
		task.Message = "任务队列已满"
		task.ErrorMessage = "任务队列已满"
		return task, false, fmt.Errorf("客户新增物料重建任务队列已满")
	}
}

// LatestRebuildTask returns the latest rebuild task.
func LatestRebuildTask(ctx context.Context, svcCtx *svc.ServiceContext) (model.MaterialDeliveryRebuildTask, error) {
	if err := markInterruptedRebuildTasksWhenIdle(ctx, svcCtx); err != nil {
		return model.MaterialDeliveryRebuildTask{}, err
	}

	var task model.MaterialDeliveryRebuildTask
	err := svcCtx.MaterialDeliveryRebuildTaskModel.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"created_at", -1}})).Decode(&task)
	if err != nil {
		return model.MaterialDeliveryRebuildTask{}, err
	}
	return task, nil
}

// ListRebuildTasks returns rebuild tasks by status and pagination.
func ListRebuildTasks(ctx context.Context, svcCtx *svc.ServiceContext, status string, page, size int64) ([]model.MaterialDeliveryRebuildTask, int64, error) {
	if err := markInterruptedRebuildTasksWhenIdle(ctx, svcCtx); err != nil {
		return nil, 0, err
	}

	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	total, err := svcCtx.MaterialDeliveryRebuildTaskModel.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("统计客户新增物料重建任务失败:%w", err)
	}
	opts := options.Find().
		SetSort(bson.D{{"created_at", -1}}).
		SetSkip((page - 1) * size).
		SetLimit(size)
	cur, err := svcCtx.MaterialDeliveryRebuildTaskModel.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("查询客户新增物料重建任务失败:%w", err)
	}
	defer cur.Close(ctx)

	var tasks []model.MaterialDeliveryRebuildTask
	if err = cur.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("解析客户新增物料重建任务失败:%w", err)
	}
	return tasks, total, nil
}

func startRebuildWorker() {
	rebuildWorkerOnce.Do(func() {
		go func() {
			for msg := range rebuildTaskQueue {
				runRebuildTask(msg)
			}
		}()
	})
}

func runRebuildTask(msg rebuildTaskMessage) {
	defer func() {
		rebuildTaskMu.Lock()
		if activeRebuildTaskId == msg.taskId.Hex() {
			activeRebuildTaskId = ""
		}
		rebuildTaskMu.Unlock()
	}()

	if err := updateRebuildTaskStatus(msg.svcCtx, msg.taskId, taskCode.MaterialDeliveryRebuildTaskStatusRunning, 0, 0, "任务正在执行", ""); err != nil {
		fmt.Printf("[Error]更新客户新增物料重建任务[%s]为执行中失败:%s\n", msg.taskId.Hex(), err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), rebuildTaskTimeout)
	defer cancel()

	progress := func(orderCount, deliveryCount int) {
		if err := updateRebuildTaskProgress(msg.svcCtx, msg.taskId, int64(orderCount), int64(deliveryCount), "任务正在执行"); err != nil {
			fmt.Printf("[Error]更新客户新增物料重建任务[%s]进度失败:%s\n", msg.taskId.Hex(), err.Error())
		}
	}
	orderCount, deliveryCount, err := RebuildCustomerMaterialDeliveries(ctx, msg.svcCtx, progress)
	if err != nil {
		if e := updateRebuildTaskStatus(msg.svcCtx, msg.taskId, taskCode.MaterialDeliveryRebuildTaskStatusFailed, int64(orderCount), int64(deliveryCount), "任务执行失败", err.Error()); e != nil {
			fmt.Printf("[Error]更新客户新增物料重建任务[%s]失败状态失败:%s\n", msg.taskId.Hex(), e.Error())
		}
		fmt.Printf("[Error]客户新增物料重建任务[%s]执行失败:%s\n", msg.taskId.Hex(), err.Error())
		return
	}

	message := fmt.Sprintf("任务执行成功，扫描出库单%d张，生成客户新增物料记录%d条", orderCount, deliveryCount)
	if err = updateRebuildTaskStatus(msg.svcCtx, msg.taskId, taskCode.MaterialDeliveryRebuildTaskStatusSuccess, int64(orderCount), int64(deliveryCount), message, ""); err != nil {
		fmt.Printf("[Error]更新客户新增物料重建任务[%s]成功状态失败:%s\n", msg.taskId.Hex(), err.Error())
	}
}

func findRebuildTaskById(ctx context.Context, svcCtx *svc.ServiceContext, taskId string) (model.MaterialDeliveryRebuildTask, error) {
	id, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return model.MaterialDeliveryRebuildTask{}, err
	}
	var task model.MaterialDeliveryRebuildTask
	if err = svcCtx.MaterialDeliveryRebuildTaskModel.FindOne(ctx, bson.M{"_id": id}).Decode(&task); err != nil {
		return model.MaterialDeliveryRebuildTask{}, fmt.Errorf("查询客户新增物料重建任务[%s]失败:%w", taskId, err)
	}
	return task, nil
}

func markInterruptedRebuildTasks(ctx context.Context, svcCtx *svc.ServiceContext) error {
	now := time.Now().Unix()
	filter := bson.M{"status": bson.M{"$in": []string{taskCode.MaterialDeliveryRebuildTaskStatusQueued, taskCode.MaterialDeliveryRebuildTaskStatusRunning}}}
	update := bson.M{"$set": bson.M{
		"status":        taskCode.MaterialDeliveryRebuildTaskStatusFailed,
		"message":       "任务已中断",
		"error_message": "服务重启或任务未被当前进程接管",
		"finished_at":   now,
		"updated_at":    now,
	}}
	if _, err := svcCtx.MaterialDeliveryRebuildTaskModel.UpdateMany(ctx, filter, update); err != nil {
		return fmt.Errorf("恢复客户新增物料重建任务状态失败:%w", err)
	}
	return nil
}

func markInterruptedRebuildTasksWhenIdle(ctx context.Context, svcCtx *svc.ServiceContext) error {
	rebuildTaskMu.Lock()
	idle := activeRebuildTaskId == ""
	rebuildTaskMu.Unlock()
	if !idle {
		return nil
	}
	return markInterruptedRebuildTasks(ctx, svcCtx)
}

func updateRebuildTaskProgress(svcCtx *svc.ServiceContext, taskId primitive.ObjectID, orderCount, deliveryCount int64, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	update := bson.M{"$set": bson.M{
		"order_count":    orderCount,
		"delivery_count": deliveryCount,
		"message":        message,
		"updated_at":     time.Now().Unix(),
	}}
	_, err := svcCtx.MaterialDeliveryRebuildTaskModel.UpdateByID(ctx, taskId, update)
	return err
}

func updateRebuildTaskStatus(svcCtx *svc.ServiceContext, taskId primitive.ObjectID, status string, orderCount, deliveryCount int64, message, errorMessage string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().Unix()
	set := bson.M{
		"status":         status,
		"order_count":    orderCount,
		"delivery_count": deliveryCount,
		"message":        message,
		"error_message":  errorMessage,
		"updated_at":     now,
	}
	switch status {
	case taskCode.MaterialDeliveryRebuildTaskStatusRunning:
		set["started_at"] = now
	case taskCode.MaterialDeliveryRebuildTaskStatusSuccess, taskCode.MaterialDeliveryRebuildTaskStatusFailed:
		set["finished_at"] = now
	}
	_, err := svcCtx.MaterialDeliveryRebuildTaskModel.UpdateByID(ctx, taskId, bson.M{"$set": set})
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	return err
}
