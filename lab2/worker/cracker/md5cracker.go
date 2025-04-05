package cracker

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strings"
	"worker/models"
)

type MD5Cracker struct {
	alphabet string
}

// brute force md5 cracker [a-z0-9]
func NewMD5Cracker() *MD5Cracker {
	letters := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	return &MD5Cracker{
		alphabet: letters + digits,
	}
}

func (c *MD5Cracker) Crack(task models.CrackTaskRequest) (string, error) {
	targetHash := strings.ToLower(task.Hash)
	base := len(c.alphabet)

	var totalCombinations int64 = 0

	for l := 1; l <= task.MaxLength; l++ {
		total := int64(math.Pow(float64(base), float64(l)))
		totalCombinations += total/int64(task.PartCount) + 1
	}

	log.Printf("Starting crack attempt for hash: %s (max length: %d, part: %d/%d)",
		task.Hash, task.MaxLength, task.PartNumber, task.PartCount)

	for length := 1; length <= task.MaxLength; length++ {
		total := int(math.Pow(float64(base), float64(length)))
		for i := 0; i < total; i++ {
			if i%task.PartCount != task.PartNumber {
				continue
			}

			candidate := c.intToCandidate(i, length)
			hashBytes := md5.Sum([]byte(candidate))
			hashCandidate := hex.EncodeToString(hashBytes[:])
			if hashCandidate == targetHash {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("solution not found")
}

// преобразуем индекс в строку используя алфавит (некая алфавитная система счисления)
func (c *MD5Cracker) intToCandidate(num int, length int) string {
	base := len(c.alphabet)
	candidate := make([]byte, length)
	for j := length - 1; j >= 0; j-- {
		candidate[j] = c.alphabet[num%base]
		num /= base
	}
	return string(candidate)
}
