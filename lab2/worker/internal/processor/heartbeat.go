package processor

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// HeartbeatUpdater обновляет поле heartbeat для подзадач, находящихся в статусе WORKING.
type HeartbeatUpdater struct {
	WorkerID   string
	Collection *mongo.Collection
}

// NewHeartbeatUpdater возвращает новый экземпляр HeartbeatUpdater.
func NewHeartbeatUpdater(workerID string, coll *mongo.Collection) *HeartbeatUpdater {
	return &HeartbeatUpdater{
		WorkerID:   workerID,
		Collection: coll,
	}
}

// Start запускает цикл обновления heartbeat каждые 10 секунд.
func (h *HeartbeatUpdater) Start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		filter := bson.M{
			"subTasks": bson.M{
				"$elemMatch": bson.M{
					"workerId": h.WorkerID,
					"status":   "WORKING",
				},
			},
		}
		update := bson.M{"$set": bson.M{
			"subTasks.$[elem].heartbeat": time.Now(),
		}}
		opts := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{bson.M{"elem.workerId": h.WorkerID, "elem.status": "WORKING"}},
		})
		_, err := h.Collection.UpdateMany(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Worker %s: ошибка обновления heartbeat: %v", h.WorkerID[:5], err)
		}
		cancel()
	}
}
