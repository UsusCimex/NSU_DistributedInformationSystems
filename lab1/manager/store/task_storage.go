package store

import "manager/models"

var GlobalTaskStorage *models.TaskStorage

func Init() {
	GlobalTaskStorage = models.NewTaskStorage()
}
