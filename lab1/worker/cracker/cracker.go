package cracker

import (
	"worker/models"
)

type Cracker interface {
	Crack(task models.CrackTaskRequest) (string, error)
}
