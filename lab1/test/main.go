package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type HashCrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type HashCrackResponse struct {
	RequestId string `json:"requestId"`
}

type CrackStatus struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func computeMD5(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  -md5 <text>             : prints MD5 hash of text")
	fmt.Println("  -crack <hash> [maxLength] : sends crack request with optional maxLength (default=3) and returns requestId")
	fmt.Println("  -status <requestId>     : fetches and prints status of crack request")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]
	switch cmd {
	case "-md5":
		if len(os.Args) < 3 {
			usage()
		}
		text := os.Args[2]
		fmt.Println(computeMD5(text))
	case "-crack":
		if len(os.Args) < 3 {
			usage()
		}
		hash := os.Args[2]
		maxLength := 3
		if len(os.Args) >= 4 {
			if v, err := strconv.Atoi(os.Args[3]); err == nil {
				maxLength = v
			} else {
				fmt.Println("Invalid maxLength provided, using default 3")
			}
		}
		reqPayload := HashCrackRequest{
			Hash:      hash,
			MaxLength: maxLength,
		}
		jsonData, err := json.Marshal(reqPayload)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}
		crackURL := "http://localhost:8080/api/hash/crack"
		resp, err := http.Post(crackURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error sending crack request:", err)
			return
		}
		defer resp.Body.Close()
		var crackResp HashCrackResponse
		if err := json.NewDecoder(resp.Body).Decode(&crackResp); err != nil {
			fmt.Println("Error decoding response:", err)
			return
		}
		fmt.Println("RequestId:", crackResp.RequestId)
	case "-status":
		if len(os.Args) < 3 {
			usage()
		}
		requestId := os.Args[2]
		statusURL := fmt.Sprintf("http://localhost:8080/api/hash/status?requestId=%s", requestId)
		resp, err := http.Get(statusURL)
		if err != nil {
			fmt.Println("Error fetching status:", err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response:", err)
			return
		}
		var status CrackStatus
		if err := json.Unmarshal(body, &status); err != nil {
			fmt.Println("Error decoding status:", err)
			return
		}
		fmt.Printf("Status: %s\n", status.Status)
		if status.Status == "IN_PROGRESS" && len(status.Data) > 0 {
			progressStr := status.Data[0]
			progressStr = strings.Trim(progressStr, "[]%")
			progress, _ := strconv.ParseFloat(progressStr, 64)
			fmt.Printf("Progress: %.1f%%\n", progress)
		}
		if status.Status == "DONE" && len(status.Data) > 0 {
			fmt.Printf("Found %d results:\n", len(status.Data))
			for i, result := range status.Data {
				fmt.Printf("  %d. %s\n", i+1, result)
			}
		} else if status.Status == "FAIL" {
			fmt.Println("No results found")
		}
	default:
		usage()
	}
}
