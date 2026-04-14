package openrouter

import (
	"encoding/json"
	"testing"
)

func TestChatResponseUsageParsing(t *testing.T) {
	payload := []byte(`{
		"id":"chatcmpl-test",
		"choices":[{"message":{"role":"assistant","content":"{\"contains_bibliography\":true}"}}],
		"usage":{
			"prompt_tokens":25,
			"completion_tokens":10,
			"total_tokens":35,
			"cost":0.00123
		}
	}`)

	var resp ChatResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Usage == nil {
		t.Fatalf("expected usage to be present")
	}
	if resp.Usage.TotalTokens != 35 {
		t.Fatalf("unexpected total tokens %d", resp.Usage.TotalTokens)
	}
	if resp.Usage.Cost == nil || *resp.Usage.Cost != 0.00123 {
		t.Fatalf("unexpected cost %#v", resp.Usage.Cost)
	}
}
