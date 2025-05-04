package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"strconv" // convert hex to decimal
	"time"
)

/* NOTES:
run continuously
track previous height in temp file (.last_height)
send Telegram message only when:
❌ Sync is stuck (.last_status)
✅ Sync resumes (block height increases again)
*/

// ethereum json-rpc response
type ethResponse struct {
	Result string `json:"result"`
}

const (
	lastHeightFile = ".last_height"
	lastStatusFile = ".last_status"
	checkInterval  = 60 * time.Second // 1 minute
)

/*
get the current height from localhost:<RPC_PORT> using JSON-RPC.
parse the response and convert the hex block height (e.g. 0xbc23a5) into decimal (e.g. 12345669).
*/
func getBlockHeight() (int64, error) {
	rpcPort := os.Getenv("RPC_PORT")
	if rpcPort == "" {
		rpcPort = "8080"
	}
	url := fmt.Sprintf("http://localhost:%s", rpcPort)
	reqBody := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("RPC call failed: %v", err)
	}
	defer resp.Body.Close()

	var result ethResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode error: %v", err)
	}

	height, err := strconv.ParseInt(result.Result[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("hex to int error: %v", err)
	}
	return height, nil
}

/*
send message to Telegram chat using Bot API.
use TELEGRAM_TOKEN and TELEGRAM_CHAT_ID from .env
*/
func sendTelegramMessage(message string) error {
	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("telegram API error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned non-OK status: %v", resp.Status)
	}
	return nil
}

// read & write integer (block height)
func readIntFromFile(file string) (int64, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return -1, err
	}
	return strconv.ParseInt(string(data), 10, 64)
}

// read & write integer (block height)
func writeIntToFile(file string, val int64) error {
	return os.WriteFile(file, []byte(strconv.FormatInt(val, 10)), 0644)
}

// read & write string (status: "ok", "stuck", "down")
func readStringFromFile(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// read & write string (status: "ok", "stuck", "down")
func writeStringToFile(file string, val string) error {
	return os.WriteFile(file, []byte(val), 0644)
}

func main() {
	_ = godotenv.Load()

	for {
		currentHeight, err := getBlockHeight()
		lastStatus, _ := readStringFromFile(lastStatusFile)
		lastHeight, _ := readIntFromFile(lastHeightFile)

		if err != nil {
			fmt.Println("❌ RPC error:", err)

			if lastStatus != "down" {
				sendTelegramMessage("🚨 Monad RPC is DOWN! Unable to connect to port.")
				writeStringToFile(lastStatusFile, "down")
			}

			time.Sleep(checkInterval)
			continue
		}

		if lastStatus == "down" {
			sendTelegramMessage(fmt.Sprintf("✅ Monad RPC is back UP! Current height: %d", currentHeight))
			writeStringToFile(lastStatusFile, "ok")
		}

		if lastHeight == currentHeight && lastStatus != "stuck" {
			msg := fmt.Sprintf("⚠️ Monad node stuck at height: %d", currentHeight)
			sendTelegramMessage(msg)
			writeStringToFile(lastStatusFile, "stuck")

		} else if lastHeight != -1 && lastHeight != currentHeight && lastStatus == "stuck" {
			msg := fmt.Sprintf("✅ Monad node syncing resumed! Current height: %d", currentHeight)
			sendTelegramMessage(msg)
			writeStringToFile(lastStatusFile, "ok")
		}

		writeIntToFile(lastHeightFile, currentHeight)
		if lastHeight != currentHeight && lastStatus != "stuck" {
			writeStringToFile(lastStatusFile, "ok")
		}

		time.Sleep(checkInterval)
	}
}
