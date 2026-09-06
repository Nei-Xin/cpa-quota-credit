package pricing

import "testing"

func TestParseExplicitLongContextRates(t *testing.T) {
	s := NewService(Config{})
	data, err := s.parseData([]byte(`{"gpt-6-astra":{
		"input_cost_per_token":0.00001,
		"input_cost_per_token_above_272k_tokens":0.00002,
		"output_cost_per_token_above_272k_tokens_priority":0.00015,
		"cache_read_input_token_cost_above_272k_tokens_flex":0.000001
	}}`))
	if err != nil {
		t.Fatal(err)
	}
	p := data["gpt-6-astra"]
	if p == nil || p.LongContext272K.Input != 20e-6 ||
		p.LongContext272K.OutputPriority != 150e-6 ||
		p.LongContext272K.CacheReadFlex != 1e-6 {
		t.Fatalf("long context rates not parsed: %+v", p)
	}
}
