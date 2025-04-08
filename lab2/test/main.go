package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	baseURL = "http://localhost:8080/api/hash"
)

type CrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type CrackResponse struct {
	RequestId string `json:"requestId"`
}

type StatusResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func main() {
	md5Command := flag.String("md5", "", "Строка для хэширования в MD5")
	crackCommand := flag.String("crack", "", "MD5 хэш для расшифровки")
	statusCommand := flag.String("status", "", "ID запроса для проверки статуса")
	autoFlag := flag.Bool("auto", false, "Автоматический переход между командами")

	flag.Parse()

	args := flag.Args()

	switch {
	case *md5Command != "":
		hash := md5.Sum([]byte(*md5Command))
		hashStr := hex.EncodeToString(hash[:])
		fmt.Printf("%s %d\n", hashStr, len(*md5Command))

		if *autoFlag {
			// Автоматически запускаем crack
			req := CrackRequest{
				Hash:      hashStr,
				MaxLength: len(*md5Command),
			}
			data, err := json.Marshal(req)
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
				os.Exit(1)
			}

			resp, err := http.Post(baseURL+"/crack", "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			var crackResp CrackResponse
			if err := json.NewDecoder(resp.Body).Decode(&crackResp); err != nil {
				fmt.Printf("Ошибка: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("ID запроса: %s\n", crackResp.RequestId)

			// Автоматически проверяем статус
			checkStatus(crackResp.RequestId)
		}

	case *crackCommand != "":
		if len(args) != 1 {
			fmt.Println("Использование: -crack <hash> <length>")
			os.Exit(1)
		}
		maxLength := 0
		if _, err := fmt.Sscanf(args[0], "%d", &maxLength); err != nil {
			fmt.Println("Ошибка: длина должна быть числом")
			os.Exit(1)
		}

		req := CrackRequest{
			Hash:      *crackCommand,
			MaxLength: maxLength,
		}
		data, err := json.Marshal(req)
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post(baseURL+"/crack", "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var crackResp CrackResponse
		if err := json.NewDecoder(resp.Body).Decode(&crackResp); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", crackResp.RequestId)

		if *autoFlag {
			// Автоматически проверяем статус
			checkStatus(crackResp.RequestId)
		}

	case *statusCommand != "":
		checkStatus(*statusCommand)

	default:
		fmt.Println("Использование:")
		fmt.Println("  -md5 <string>         Хэширование строки в MD5")
		fmt.Println("  -crack <hash> <length> Расшифровка MD5 хэша")
		fmt.Println("  -status <id>          Проверка статуса расшифровки")
		fmt.Println("  -auto                 Автоматический переход между командами")
	}
}

func checkStatus(requestId string) {
	for {
		resp, err := http.Get(fmt.Sprintf("%s/status?requestId=%s", baseURL, requestId))
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}

		var statusResp StatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			resp.Body.Close()
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}
		resp.Body.Close()

		switch statusResp.Status {
		case "DONE":
			fmt.Printf("Результат: %v\n", statusResp.Data)
			return
		case "ERROR":
			fmt.Printf("Ошибка: %v\n", statusResp.Data)
			return
		case "IN_PROGRESS":
			progress, ok := statusResp.Data.(float64)
			if !ok {
				fmt.Printf("Ошибка формата прогресса\n")
				return
			}
			fmt.Printf("Прогресс: %.2f%%\n", progress)
			time.Sleep(1 * time.Second)
		default:
			fmt.Printf("Неизвестный статус: %s\n", statusResp.Status)
			return
		}
	}
}
