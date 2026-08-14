package console_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGuidesRejectsDuplicateSlugAndUnsafeContent(t *testing.T) {
	tests := []struct {
		name    string
		guides  string
		message string
	}{
		{
			name: "duplicate slug",
			guides: `[
				{"id":1,"slug":"quick-start","title":"One","summary":"","content":"Body","format":"markdown","updatedAt":"2026-08-14T00:00:00Z","order":1,"published":true},
				{"id":2,"slug":"quick-start","title":"Two","summary":"","content":"Body","format":"html","updatedAt":"2026-08-14T00:00:00Z","order":2,"published":true}
			]`,
			message: "duplicate slug",
		},
		{
			name:    "unsafe content",
			guides:  `[{"id":1,"slug":"unsafe","title":"Unsafe","summary":"","content":"<img src=x onerror=alert(1)>","format":"html","updatedAt":"2026-08-14T00:00:00Z","order":1,"published":true}]`,
			message: "unsafe HTML",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateGuides(test.guides)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}

func TestGetPublishedGuidesFiltersAndSorts(t *testing.T) {
	settings := GetConsoleSetting()
	original := settings.Guides
	t.Cleanup(func() {
		settings.Guides = original
	})

	settings.Guides = `[
		{"id":1,"slug":"later","title":"Later","summary":"","content":"Body","format":"markdown","updatedAt":"2026-08-14T00:00:00Z","order":20,"published":true},
		{"id":2,"slug":"hidden","title":"Hidden","summary":"","content":"Body","format":"markdown","updatedAt":"2026-08-14T00:00:00Z","order":1,"published":false},
		{"id":3,"slug":"first","title":"First","summary":"","content":"Body","format":"html","updatedAt":"2026-08-13T00:00:00Z","order":10,"published":true}
	]`

	guides := GetPublishedGuides()
	require.Len(t, guides, 2)
	assert.Equal(t, "first", guides[0].Slug)
	assert.Equal(t, "later", guides[1].Slug)
}
