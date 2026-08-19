package controller

import "testing"

func TestGuideMediaTypeValidation(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		declared  string
		maxBytes  int64
		wantType  string
		wantValid bool
	}{
		{"png", "guide.PNG", "image/png", guideImageMaxBytes, "image/png", true},
		{"video", "demo.mp4", "video/mp4; charset=binary", guideVideoMaxBytes, "video/mp4", true},
		{"mismatch", "guide.png", "image/jpeg", 0, "", false},
		{"unknown extension", "guide.exe", "application/octet-stream", 0, "", false},
		{"missing type", "guide.webp", "", 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotMax, gotValid := guideMediaType(tt.filename, tt.declared)
			if gotValid != tt.wantValid || gotType != tt.wantType || gotMax != tt.maxBytes {
				t.Fatalf("guideMediaType() = (%q, %d, %t), want (%q, %d, %t)", gotType, gotMax, gotValid, tt.wantType, tt.maxBytes, tt.wantValid)
			}
		})
	}
}
