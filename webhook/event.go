package webhook

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func generateEventID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// SendEvents は各受信者に対してイベント Webhook を非同期で POST する。
// eventType は "processed" または "delivered"。response は "delivered" 時のみ設定する。
// customArgs のキーはイベントのトップレベルに展開される（予約キーは上書きしない）。
func SendEvents(messageID string, toEmails []string, eventType string, response string, customArgs map[string]string) {
	webhookURL := os.Getenv("SENDGRID_DEV_EVENT_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	go func() {
		now := time.Now().Unix()
		smtpID := fmt.Sprintf("<%s@sendgrid-dev>", messageID)

		events := make([]map[string]interface{}, 0, len(toEmails))
		for _, addr := range toEmails {
			payload := map[string]interface{}{
				"email":         addr,
				"timestamp":     now,
				"smtp-id":       smtpID,
				"event":         eventType,
				"sg_event_id":   generateEventID(),
				"sg_message_id": messageID,
			}
			if response != "" {
				payload["response"] = response
			}
			// custom_args をトップレベルに展開（予約キーと衝突する場合は予約キー優先）
			for k, v := range customArgs {
				if _, exists := payload[k]; !exists {
					payload[k] = v
				}
			}
			events = append(events, payload)
		}

		body, err := json.Marshal(events)
		if err != nil {
			fmt.Println("Event webhook marshal error:", err)
			return
		}

		postWebhook(webhookURL, body)
	}()
}

func postWebhook(webhookURL string, body []byte) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		fmt.Println("Event webhook リクエスト生成エラー:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if key := loadSigningKey(); key != nil {
		sig, err := sign(key, timestamp, body)
		if err != nil {
			fmt.Println("Event webhook 署名エラー:", err)
		} else {
			req.Header.Set("X-Twilio-Email-Event-Webhook-Signature", sig)
			req.Header.Set("X-Twilio-Email-Event-Webhook-Timestamp", timestamp)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Event webhook 送信エラー:", err)
		return
	}
	defer resp.Body.Close()
}

// loadSigningKey は SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY（Base64 DER形式）から ECDSA P-256 秘密鍵を読み込む。
// 環境変数が空の場合は nil を返す。
func loadSigningKey() *ecdsa.PrivateKey {
	keyStr := os.Getenv("SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY")
	if keyStr == "" {
		return nil
	}
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyStr))
	if err != nil {
		fmt.Println("Event webhook: 署名鍵の Base64 デコード失敗:", err)
		return nil
	}
	key, err := x509.ParseECPrivateKey(der)
	if err != nil {
		fmt.Println("Event webhook: 署名鍵のパース失敗:", err)
		return nil
	}
	return key
}

// sign は SHA-256(timestamp + payload) に対して ECDSA 署名を行い、base64 DER を返す。
func sign(key *ecdsa.PrivateKey, timestamp string, payload []byte) (string, error) {
	data := append([]byte(timestamp), payload...)
	hash := sha256.Sum256(data)

	r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
	if err != nil {
		return "", err
	}

	type ecdsaSig struct {
		R, S *big.Int
	}
	der, err := asn1.Marshal(ecdsaSig{r, s})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(der), nil
}
