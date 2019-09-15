package loopguard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeHTTP(t *testing.T) {
	testCases := []struct {
		desc            string
		token           string
		incomingHeaders map[string]string

		expectedBlocked bool
		expectedHeaders map[string]string
	}{
		{
			desc:            "no header",
			token:           "token",
			incomingHeaders: map[string]string{},

			expectedBlocked: false,
			expectedHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "token",
			},
		},
		{
			desc:            "no token",
			token:           "",
			incomingHeaders: map[string]string{},

			expectedBlocked: false,
			expectedHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "",
			},
		},
		{
			desc:  "other tokens",
			token: "token",
			incomingHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo",
			},

			expectedBlocked: false,
			expectedHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo, token",
			},
		},
		{
			desc:  "other tokens with similar name",
			token: "token",
			incomingHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo, almost token, ",
			},

			expectedBlocked: false,
			expectedHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo, almost token, token",
			},
		},
		{
			desc:  "previous token",
			token: "token",
			incomingHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "token",
			},

			expectedBlocked: true,
		},
		{
			desc:  "previous token in chain",
			token: "token",
			incomingHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo, token, bar",
			},

			expectedBlocked: true,
		},
		{
			desc:  "previous token in malformed chain",
			token: "token",
			incomingHeaders: map[string]string{
				"X-HttpBroadcast-Guard": "foo,    token    , bar",
			},

			expectedBlocked: true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(http.MethodGet, "", nil)
			require.NoError(t, err)

			for k, v := range test.incomingHeaders {
				req.Header.Set(k, v)
			}

			var nextRequest *http.Request
			m := NewLoopGuard(test.token, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				nextRequest = r
			}))

			rr := httptest.NewRecorder()
			m.ServeHTTP(rr, req)

			if test.expectedBlocked {
				assert.Nil(t, nextRequest)
				assert.Equal(t, 202, rr.Code)
			} else {
				assert.Equal(t, req, nextRequest)
				for k, v := range test.expectedHeaders {
					assert.Equal(t, v, nextRequest.Header.Get(k))
				}
			}
		})
	}
}
