package immich

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Test helper functions and types
************************************************************************************************/

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		apiURL        string
		resetStacks   bool
		replaceStacks bool
		dryRun        bool
		wantErr       bool
	}{
		{
			name:          "valid config",
			apiKey:        "test-key",
			apiURL:        "http://test.com",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       false,
		},
		{
			name:          "missing api key",
			apiKey:        "",
			apiURL:        "http://test.com",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       true,
		},
		{
			name:          "missing api url",
			apiKey:        "test-key",
			apiURL:        "",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			client := NewClient(tt.apiURL, tt.apiKey, tt.resetStacks, tt.replaceStacks, tt.dryRun, true, false, logrus.New())

			// Assert
			if tt.wantErr {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
				assert.Equal(t, tt.apiKey, client.apiKey)
				assert.Equal(t, tt.apiURL+"/api", client.apiURL)
			}
		})
	}
}

func TestFetchAssets(t *testing.T) {
	tests := []struct {
		name      string
		client    *Client
		size      int
		stacksMap map[string]utils.TStack
		wantErr   bool
	}{
		{
			name: "invalid client",
			client: &Client{
				apiKey: "invalid",
				apiURL: "invalid",
				logger: logrus.New(),
				client: &http.Client{},
			},
			size:      10,
			stacksMap: make(map[string]utils.TStack),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			assets, err := tt.client.FetchAssets(tt.size, tt.stacksMap)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, assets)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, assets)
			}
		})
	}
}

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestModifyStack(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		assets  []string
		wantErr bool
	}{
		{
			name: "empty assets",
			client: &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logrus.New(),
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			},
			assets:  []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.client.ModifyStack(tt.assets)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
