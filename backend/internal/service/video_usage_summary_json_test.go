package service

import (
	"encoding/json"
	"testing"
)

func TestVideoUsageSummaryJSONUsesStableSnakeCasePricingFields(t *testing.T) {
	payload, err := json.Marshal(VideoUsageSummary{
		Provider:       VideoProviderSeedance,
		Model:          "doubao-seedance-2-0-260128",
		Status:         VideoStatusSucceeded,
		Count:          1,
		CostEstimate:   5.0094,
		Duration:       5,
		Currency:       BillingCurrencyCNY,
		PricingSource:  PricingSourceProviderUsage,
		PricingVersion: VideoPricingVersionSeedance202603,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"provider", "model", "status", "count", "cost_estimate", "duration", "currency", "pricing_source", "pricing_version"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing JSON contract key %q in %s", key, payload)
		}
	}
	for _, legacy := range []string{"Provider", "CostEstimate", "PricingSource"} {
		if _, ok := got[legacy]; ok {
			t.Fatalf("legacy PascalCase key %q leaked into %s", legacy, payload)
		}
	}
}
