package api

import (
	"math"
	"testing"
)

func TestThrowTypeRequiresEvidenceForVelocityFallback(t *testing.T) {
	tests := []struct {
		name    string
		utility Utility
		want    UtilityThrowType
	}{
		{"death release without priming", Utility{InitialVelocityZ: -1}, UtilityThrowTypeUnknown},
		{"primed death release", Utility{ThrowStrength: 0.959375, InitialVelocityZ: -1}, UtilityThrowTypeUnknown},
		{"missing velocity", Utility{}, UtilityThrowTypeUnknown},
		{"only inherited player motion", Utility{InitialVelocityX: 100, ThrowerVelocityX: 80}, UtilityThrowTypeUnknown},
		{"nonfinite velocity", Utility{InitialVelocityX: math.Inf(1)}, UtilityThrowTypeUnknown},
		{"nonfinite angle", Utility{InitialVelocityX: 200, ThrowerPitch: float32(math.NaN())}, UtilityThrowTypeUnknown},
		{"known full strength", Utility{ThrowStrength: 1}, UtilityThrowTypeLeftClick},
		{"known half strength", Utility{ThrowStrength: 0.5}, UtilityThrowTypeDoubleClick},
		{"fallback soft throw", Utility{InitialVelocityX: 200}, UtilityThrowTypeRightClick},
		{"fallback medium throw", Utility{InitialVelocityX: 400}, UtilityThrowTypeDoubleClick},
		{"fallback full throw", Utility{InitialVelocityX: 600}, UtilityThrowTypeLeftClick},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyThrowTypeByStrength(&test.utility); got != test.want {
				t.Fatalf("throw type = %q, want %q", got, test.want)
			}
		})
	}
}
