package registration

import (
	"log"
	"os"
	"time"
	"worker/config"

	"common/utils"
)

func RegisterWithManager(cfg *config.Config) error {
	registration := struct {
		WorkerURL  string `json:"workerUrl"`
		MaxWorkers int    `json:"maxWorkers"`
	}{
		WorkerURL:  os.Getenv("WORKER_URL"),
		MaxWorkers: cfg.MaxWorkers,
	}

	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}

	sendCfg := utils.SendConfig{
		MaxRetries:      10,
		Delay:           5 * time.Second,
		SuccessStatus:   200,
		RetryOnStatuses: []int{500, 502, 503, 504},
	}

	req := utils.SendRequest{
		URL:     managerURL + "/internal/api/worker/register",
		Payload: registration,
	}

	if err := utils.RetryingSend(req, sendCfg); err != nil {
		return err
	}

	log.Printf("Successfully registered with manager at %s", managerURL)
	return nil
}
