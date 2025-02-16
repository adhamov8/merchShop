package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.New(rand.NewSource((time.Now().UnixNano())))
}

type authResponse struct {
	Token string `json:"token"`
}

type InfoResponse struct {
	Coins     int `json:"coins"`
	Inventory []struct {
		Type     string `json:"type"`
		Quantity int    `json:"quantity"`
	} `json:"inventory"`
	CoinHistory struct {
		Received []struct {
			FromUser string `json:"fromUser"`
			Amount   int    `json:"amount"`
		} `json:"received"`
		Sent []struct {
			ToUser string `json:"toUser"`
			Amount int    `json:"amount"`
		} `json:"sent"`
	} `json:"coinHistory"`
}

func TestFullScenario(t *testing.T) {
	time.Sleep(2 * time.Second)

	ZiyoUsername := fmt.Sprintf("ZiyoE2E_%d", rand.Int31())
	AlibekUsername := fmt.Sprintf("AlibekE2E_%d", rand.Int31())

	ZiyoToken, err := registerOrLogin(ZiyoUsername, "Strong@Pass123")
	assert.NoError(t, err)
	assert.NotEmpty(t, ZiyoToken)

	AlibekToken, err := registerOrLogin(AlibekUsername, "Strong@Pass123")
	assert.NoError(t, err)
	assert.NotEmpty(t, AlibekToken)

	err = buyItem(ZiyoToken, "book")
	assert.NoError(t, err, "error when buying book")

	err = sendCoins(ZiyoToken, AlibekUsername, 100)
	assert.NoError(t, err, "error sending coins")

	ZiyoInfo, err := getInfo(ZiyoToken)
	assert.NoError(t, err)
	assert.Equal(t, 850, ZiyoInfo.Coins, "Ziyo's coins must be 850 after book(50) + send(100)")

	Alibek, err := getInfo(AlibekToken)
	assert.NoError(t, err)
	assert.Equal(t, 1100, Alibek.Coins, "Alibek's coins must be 1100 after receiving 100")
}

func registerOrLogin(username, password string) (string, error) {
	reqBody := map[string]string{"username": username, "password": password}
	data, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:8080/api/auth", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", parseError(resp)
	}
	var ar authResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return "", err
	}
	return ar.Token, nil
}

func buyItem(token, itemName string) error {
	url := "http://localhost:8080/api/buy/" + itemName
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

func sendCoins(token, toUser string, amount int) error {
	reqBody := map[string]interface{}{"toUser": toUser, "amount": amount}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/api/sendCoin", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return parseError(resp)
	}
	return nil
}

func getInfo(token string) (*InfoResponse, error) {
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/api/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, parseError(resp)
	}
	var result InfoResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	return &result, err
}

func parseError(resp *http.Response) error {
	var errBody map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if e, ok := errBody["errors"].(string); ok {
		return &ErrAPI{status: resp.StatusCode, msg: e}
	}
	return &ErrAPI{status: resp.StatusCode, msg: "unknown error"}
}

type ErrAPI struct {
	status int
	msg    string
}

func (e *ErrAPI) Error() string {
	return "API error: status=" + strconv.Itoa(e.status) + ", msg=" + e.msg
}
