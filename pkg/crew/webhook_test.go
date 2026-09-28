package crew

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/webhook"
)

func TestKickoff_FiresWebhook(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "1")
	var gotEvent, gotSig string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		gotBody, gotEvent, gotSig = body, r.Header.Get("X-Gocrew-Event"), r.Header.Get("X-Gocrew-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	agent := &agents.Agent{Role: "worker", LLM: &mockLLM{}}
	agent.I18N, _ = i18n.NewI18N("en")
	task := &tasks.Task{Description: "job", Agent: agent}
	task.I18N, _ = i18n.NewI18N("en")
	c := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	c.WebhookURL = srv.URL
	c.WebhookSecret = "hook-secret"

	if _, err := c.Kickoff(context.Background()); err != nil {
		t.Fatalf("kickoff: %v", err)
	}
	if gotEvent != "kickoff.completed" {
		t.Errorf("event = %q", gotEvent)
	}
	var p webhook.Payload
	if err := json.Unmarshal(gotBody, &p); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if p.Status != "completed" {
		t.Errorf("status = %q", p.Status)
	}
	if err := webhook.VerifySignature("hook-secret", gotBody, gotSig, "", 0); err != nil {
		t.Errorf("signature: %v", err)
	}
}

func TestKickoff_BadWebhookURLDoesNotFail(t *testing.T) {
	agent := &agents.Agent{Role: "worker", LLM: &mockLLM{}}
	agent.I18N, _ = i18n.NewI18N("en")
	task := &tasks.Task{Description: "job", Agent: agent}
	task.I18N, _ = i18n.NewI18N("en")
	c := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	c.WebhookURL = "http://169.254.169.254/hook" // blocked by SSRF gate
	c.WebhookSecret = "s"
	if _, err := c.Kickoff(context.Background()); err != nil {
		t.Fatalf("kickoff must succeed despite bad webhook: %v", err)
	}
}
