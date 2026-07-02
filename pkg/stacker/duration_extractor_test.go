package stacker

import (
	"encoding/json"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationExtractor(t *testing.T) {
	extractor, ok := getExtractor("duration")
	require.True(t, ok, "duration extractor must be registered")

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "v2 string", body: `{"id":"1","duration":"0:00:12.500000"}`, want: "0:00:12.500000"},
		{name: "v3 number", body: `{"id":"2","duration":1250}`, want: "1250"},
		{name: "v3 null", body: `{"id":"3","duration":null}`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a utils.TAsset
			require.NoError(t, json.Unmarshal([]byte(tt.body), &a))

			got, err := extractor(a, utils.TCriteria{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
