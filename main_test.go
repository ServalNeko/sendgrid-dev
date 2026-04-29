package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"github.com/ServalNeko/sendgrid-dev/route"
	"github.com/steinfletcher/apitest"
)

func TestSend(t *testing.T) {
	os.Setenv("SENDGRID_DEV_TEST", "1")

	// NG (Not POST)
	apitest.New().
		Handler(route.Init()).
		Get("/v3/mail/send").
		Expect(t).
		Body(`{"errors":[{"message":"POST method allowed only","field":null,"help":null}]}`).
		Status(http.StatusMethodNotAllowed).
		End()

	// NG (Missing Authorization)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Expect(t).
		Body(`{"errors":[{"message":"The provided authorization grant is invalid, expired, or revoked","field":null,"help":null}]}`).
		Status(http.StatusUnsupportedMediaType).
		End()

	// NG (Missing Content-Type)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		Expect(t).
		Body(`{"errors":[{"message":"Content-Type should be application/json","field":null,"help":null}]}`).
		Status(http.StatusUnsupportedMediaType).
		End()

	// NG (Content-Type is not application/json)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		Headers(map[string]string{"Content-Type": "text/plain"}).
		Expect(t).
		Body(`{"errors":[{"message":"Content-Type should be application/json","field":null,"help":null}]}`).
		Status(http.StatusUnsupportedMediaType).
		End()

	// NG (Missing PostData)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(``).
		Expect(t).
		Body(`{"errors":[{"message":"Bad Request","field":null,"help":null}]}`).
		Status(http.StatusBadRequest).
		End()

	// NG (Missing personalizations)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"The personalizations field is required and must have at least one personalization.","field":"personalizations","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#-Personalizations-Errors"}]}`).
		Status(http.StatusBadRequest).
		End()

	// NG (Missing from.Email)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"The from object must be provided for every email send. It is an object that requires the email parameter, but may also contain a name parameter.  e.g. {\"email\" : \"example@example.com\"}  or {\"email\" : \"example@example.com\", \"name\" : \"Example Recipient\"}.","field":"from.email","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.from"}]}`).
		Status(http.StatusBadRequest).
		End()

	// NG (Missing subject)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"The subject is required. You can get around this requirement if you use a template with a subject defined or if every personalization has a subject defined.","field":"subject","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.subject"}]}`).
		Status(http.StatusBadRequest).
		End()

	// NG (Missing content)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject"
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"Unless a valid template_id is provided, the content parameter is required. There must be at least one defined content block. We typically suggest both text/plain and text/html blocks are included, but only one block is required.","field":"content","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.content"}]}`).
		Status(http.StatusBadRequest).
		End()

	// OK
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (subject in personalizations)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}],
				"subject": "Subject"
			}],
			"from": {
				"email": "from@example.com"
			},
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (duplicate subject (personalization priority))
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}],
				"subject": "Subject"
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple to, cc, bcc and reply-to with name)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to1@example.com",
					"name": "ToName1"
				}, {
					"email": "to2@example.com",
					"name": "ToName2"
				}],
				"cc": [{
					"email": "cc1@example.com",
					"name": "CcName1"
				}, {
					"email": "cc2@example.com",
					"name": "CcName2"
				}],
				"bcc": [{
					"email": "bcc1@example.com",
					"name": "BccName1"
				}, {
					"email": "bcc2@example.com",
					"name": "BccName2"
				}]
			}],
			"from": {
				"email": "from@example.com",
				"name": "FromName"
			},
			"reply_to": {
				"email": "reply_to@example.com",
				"name": "ReplyToName"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple personalizations)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [
				{
					"to": [{
						"email": "to1@example.com",
						"name": "ToName1"
					}, {
						"email": "to2@example.com",
						"name": "ToName2"
					}]
				},
				{
					"to": [{
						"email": "to3@example.com",
						"name": "ToName3"
					}, {
						"email": "to4@example.com",
						"name": "ToName4"
					}]
				}
			],
			"from": {
				"email": "from@example.com",
				"name": "FromName"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple personalizations with subject)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [
				{
					"to": [{
						"email": "to1@example.com",
						"name": "ToName1"
					}, {
						"email": "to2@example.com",
						"name": "ToName2"
					}],
					"subject": "Test Subject1"
				},
				{
					"to": [{
						"email": "to3@example.com",
						"name": "ToName3"
					}, {
						"email": "to4@example.com",
						"name": "ToName4"
					}],
					"subject": "Test Subject2"
				}
			],
			"from": {
				"email": "from@example.com",
				"name": "FromName"
			},
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple personalizations with substitutions)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [
				{
					"to": [{
						"email": "to1@example.com",
						"name": "ToName1"
					}],
					"substitutions": {
						"-name-": "to1"
					}
				},
				{
					"to": [{
						"email": "to2@example.com",
						"name": "ToName2"
					}],
					"substitutions": {
						"-name-": "to2"
					}
				}
			],
			"from": {
				"email": "from@example.com",
				"name": "FromName"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content -name-"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (text/html)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/html",
				"value": "<h1>Content</h1>"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple content text/plain, text/html)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content1"
			}, {
				"type": "text/html",
				"value": "<h1>Content2</h1>"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (attachements)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"attachments": [{
				"content": "dGVzdA==",
				"type": "text/plain",
				"filename": "attachment.txt"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (multiple attachements)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"attachments": [{
				"content": "dGVzdA==",
				"type": "text/plain",
				"filename": "attachment1.txt"
			}, {
				"content": "dGVzdA==",
				"type": "text/plain",
				"filename": "attachment2.txt"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// NG (attachements content is not BASE64)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"attachments": [{
				"content": "NOT BASE64",
				"type": "text/plain",
				"filename": "attachment.txt"
			}]
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"The attachment content must be base64 encoded.","field":"attachments.0.content","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.attachments.content"}]}`).
		Status(http.StatusBadRequest).
		End()

	// NG (attachements content is not BASE64 in multiple)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"attachments": [{
				"content": "dGVzdA==",
				"type": "text/plain",
				"filename": "attachment1.txt"
			}, {
				"content": "NOT_BASE64",
				"type": "text/plain",
				"filename": "attachment2.txt"
			}]
		}`).
		Expect(t).
		Body(`{"errors":[{"message":"The attachment content must be base64 encoded.","field":"attachments.1.content","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.attachments.content"}]}`).
		Status(http.StatusBadRequest).
		End()

	// OK (with SMTP Auth)
	os.Setenv("SENDGRID_DEV_SMTP_USERNAME", "username@example.com")
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}]
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (message-level custom_args)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}]
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"custom_args": {
				"userid": "1234",
				"env": "test"
			}
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()

	// OK (personalization-level custom_args overrides message-level)
	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{
				"to": [{
					"email": "to@example.com"
				}],
				"custom_args": {
					"userid": "9999"
				}
			}],
			"from": {
				"email": "from@example.com"
			},
			"subject": "Subject",
			"content": [{
				"type": "text/plain",
				"value": "Content"
			}],
			"custom_args": {
				"userid": "1234",
				"env": "test"
			}
		}`).
		Expect(t).
		Body(``).
		Status(http.StatusAccepted).
		HeaderPresent("X-Message-Id").
		End()
}

func TestEventWebhook(t *testing.T) {
	os.Setenv("SENDGRID_DEV_TEST", "1")
	defer os.Unsetenv("SENDGRID_DEV_EVENT_WEBHOOK_URL")

	// processed イベントが受信されること
	t.Run("processed イベント", func(t *testing.T) {
		received := make(chan []byte, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			received <- body
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		os.Setenv("SENDGRID_DEV_EVENT_WEBHOOK_URL", srv.URL)

		apitest.New().
			Handler(route.Init()).
			Post("/v3/mail/send").
			Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
			JSON(`{
				"personalizations": [{
					"to": [{"email": "to@example.com"}]
				}],
				"from": {"email": "from@example.com"},
				"subject": "Subject",
				"content": [{"type": "text/plain", "value": "Content"}]
			}`).
			Expect(t).
			Status(http.StatusAccepted).
			End()

		select {
		case body := <-received:
			var events []map[string]interface{}
			if err := json.Unmarshal(body, &events); err != nil {
				t.Fatalf("Webhook ペイロードのパース失敗: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("イベント数: got %d, want 1", len(events))
			}
			if events[0]["event"] != "processed" {
				t.Errorf("event: got %v, want processed", events[0]["event"])
			}
			if events[0]["email"] != "to@example.com" {
				t.Errorf("email: got %v, want to@example.com", events[0]["email"])
			}
			if events[0]["sg_message_id"] == "" {
				t.Error("sg_message_id が空")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("2秒以内に Webhook が受信されなかった")
		}
	})

	// custom_args がトップレベルに展開されること
	t.Run("custom_args 展開", func(t *testing.T) {
		received := make(chan []byte, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			received <- body
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		os.Setenv("SENDGRID_DEV_EVENT_WEBHOOK_URL", srv.URL)

		apitest.New().
			Handler(route.Init()).
			Post("/v3/mail/send").
			Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
			JSON(`{
				"personalizations": [{
					"to": [{"email": "to@example.com"}],
					"custom_args": {"userid": "9999"}
				}],
				"from": {"email": "from@example.com"},
				"subject": "Subject",
				"content": [{"type": "text/plain", "value": "Content"}],
				"custom_args": {"userid": "1234", "env": "test"}
			}`).
			Expect(t).
			Status(http.StatusAccepted).
			End()

		select {
		case body := <-received:
			var events []map[string]interface{}
			if err := json.Unmarshal(body, &events); err != nil {
				t.Fatalf("Webhook ペイロードのパース失敗: %v", err)
			}
			// personalization の custom_args がメッセージレベルを上書き
			if events[0]["userid"] != "9999" {
				t.Errorf("userid: got %v, want 9999", events[0]["userid"])
			}
			// メッセージレベルのみのキーは引き継がれる
			if events[0]["env"] != "test" {
				t.Errorf("env: got %v, want test", events[0]["env"])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("2秒以内に Webhook が受信されなかった")
		}
	})

	// 複数受信者に対してそれぞれイベントが生成されること
	t.Run("複数受信者", func(t *testing.T) {
		received := make(chan []byte, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			received <- body
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		os.Setenv("SENDGRID_DEV_EVENT_WEBHOOK_URL", srv.URL)

		apitest.New().
			Handler(route.Init()).
			Post("/v3/mail/send").
			Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
			JSON(`{
				"personalizations": [{
					"to": [
						{"email": "to1@example.com"},
						{"email": "to2@example.com"}
					]
				}],
				"from": {"email": "from@example.com"},
				"subject": "Subject",
				"content": [{"type": "text/plain", "value": "Content"}]
			}`).
			Expect(t).
			Status(http.StatusAccepted).
			End()

		select {
		case body := <-received:
			var events []map[string]interface{}
			if err := json.Unmarshal(body, &events); err != nil {
				t.Fatalf("Webhook ペイロードのパース失敗: %v", err)
			}
			if len(events) != 2 {
				t.Fatalf("イベント数: got %d, want 2", len(events))
			}
		case <-time.After(2 * time.Second):
			t.Fatal("2秒以内に Webhook が受信されなかった")
		}
	})
}

func TestSignedEventWebhook(t *testing.T) {
	os.Setenv("SENDGRID_DEV_TEST", "1")
	defer os.Unsetenv("SENDGRID_DEV_EVENT_WEBHOOK_URL")
	defer os.Unsetenv("SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY")

	// テスト用 ECDSA P-256 鍵ペアを生成
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("鍵生成失敗: %v", err)
	}
	privBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		t.Fatalf("秘密鍵シリアライズ失敗: %v", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	os.Setenv("SENDGRID_DEV_EVENT_WEBHOOK_SIGNING_KEY", string(pemBlock))

	type capturedReq struct {
		body      []byte
		signature string
		timestamp string
	}
	received := make(chan capturedReq, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- capturedReq{
			body:      body,
			signature: r.Header.Get("X-Twilio-Email-Event-Webhook-Signature"),
			timestamp: r.Header.Get("X-Twilio-Email-Event-Webhook-Timestamp"),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	os.Setenv("SENDGRID_DEV_EVENT_WEBHOOK_URL", srv.URL)

	apitest.New().
		Handler(route.Init()).
		Post("/v3/mail/send").
		Headers(map[string]string{"Authorization": "Bearer " + os.Getenv("SENDGRID_DEV_API_KEY")}).
		JSON(`{
			"personalizations": [{"to": [{"email": "to@example.com"}]}],
			"from": {"email": "from@example.com"},
			"subject": "Subject",
			"content": [{"type": "text/plain", "value": "Content"}]
		}`).
		Expect(t).
		Status(http.StatusAccepted).
		End()

	select {
	case req := <-received:
		if req.signature == "" {
			t.Error("X-Twilio-Email-Event-Webhook-Signature ヘッダーがない")
		}
		if req.timestamp == "" {
			t.Error("X-Twilio-Email-Event-Webhook-Timestamp ヘッダーがない")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("2秒以内に Signed Webhook が受信されなかった")
	}
}
