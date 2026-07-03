package utils

import (
	"encoding/json"
	"testing"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TDuration
		wantErr bool
	}{
		{name: "v2 string zero", input: `"0:00:00.000000"`, want: "0:00:00.000000"},
		{name: "v2 string non-zero", input: `"0:00:12.500000"`, want: "0:00:12.500000"},
		{name: "v3 number ms", input: `1250`, want: "1250"},
		{name: "v3 number zero", input: `0`, want: "0"},
		{name: "v3 number large", input: `32420`, want: "32420"},
		{name: "v3 number fractional", input: `1250.5`, want: "1250.5"},
		{name: "null", input: `null`, want: ""},
		{name: "empty string", input: `""`, want: ""},
		{name: "bool is invalid", input: `true`, wantErr: true},
		{name: "object is invalid", input: `{}`, wantErr: true},
		{name: "array is invalid", input: `[1]`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d TDuration
			err := json.Unmarshal([]byte(tt.input), &d)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error unmarshaling %q, got none (value=%q)", tt.input, d)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error unmarshaling %q: %v", tt.input, err)
			}
			if d != tt.want {
				t.Fatalf("unmarshaling %q = %q, want %q", tt.input, d, tt.want)
			}
		})
	}
}

func TestTAssetDurationV2AndV3(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantDuration TDuration
	}{
		{name: "v2 image string duration", body: `{"id":"a1","type":"IMAGE","duration":"0:00:00.000000"}`, wantDuration: "0:00:00.000000"},
		{name: "v3 image null duration", body: `{"id":"a2","type":"IMAGE","duration":null}`, wantDuration: ""},
		{name: "v3 animated gif number duration", body: `{"id":"a3","type":"IMAGE","duration":1250}`, wantDuration: "1250"},
		{name: "v3 video number duration", body: `{"id":"a4","type":"VIDEO","duration":32420}`, wantDuration: "32420"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a TAsset
			if err := json.Unmarshal([]byte(tt.body), &a); err != nil {
				t.Fatalf("decoding asset: %v", err)
			}
			if a.Duration != tt.wantDuration {
				t.Fatalf("Duration = %q, want %q", a.Duration, tt.wantDuration)
			}
		})
	}
}

func TestSearchResponseDecodeMixedDurations(t *testing.T) {
	body := `{"assets":{"items":[
		{"id":"1","type":"IMAGE","duration":null},
		{"id":"2","type":"IMAGE","duration":"0:00:00.000000"},
		{"id":"3","type":"IMAGE","duration":1250},
		{"id":"4","type":"VIDEO","duration":32420}
	],"nextPage":"2"}}`

	var resp TSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding search response: %v", err)
	}
	if got := len(resp.Assets.Items); got != 4 {
		t.Fatalf("decoded %d items, want 4", got)
	}
	want := []TDuration{"", "0:00:00.000000", "1250", "32420"}
	for i, w := range want {
		if resp.Assets.Items[i].Duration != w {
			t.Fatalf("item %d Duration = %q, want %q", i, resp.Assets.Items[i].Duration, w)
		}
	}
	if resp.Assets.NextPage != "2" {
		t.Fatalf("NextPage = %q, want %q", resp.Assets.NextPage, "2")
	}
}
