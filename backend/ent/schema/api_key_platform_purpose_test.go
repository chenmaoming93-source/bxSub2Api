package schema

import "testing"

func TestAPIKeyPlatformPurposeFields(t *testing.T) {
	fields := APIKey{}.Fields()
	byName := make(map[string]bool, len(fields))
	var platformOptional, platformNillable bool
	var platformMaxLen int
	var purposeDefault any
	var purposeMaxLen int
	for _, item := range fields {
		desc := item.Descriptor()
		byName[desc.Name] = true
		switch desc.Name {
		case "platform":
			platformOptional = desc.Optional
			platformNillable = desc.Nillable
			platformMaxLen = desc.Size
		case "purpose":
			purposeDefault = desc.Default
			purposeMaxLen = desc.Size
		}
	}

	if !byName["platform"] || !platformOptional || !platformNillable || platformMaxLen != 50 {
		t.Fatalf("platform field mismatch: exists=%v optional=%v nillable=%v maxLen=%d", byName["platform"], platformOptional, platformNillable, platformMaxLen)
	}
	if !byName["purpose"] || purposeDefault != "user_created" || purposeMaxLen != 20 {
		t.Fatalf("purpose field mismatch: exists=%v default=%v maxLen=%d", byName["purpose"], purposeDefault, purposeMaxLen)
	}
}
