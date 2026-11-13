package accesslog

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonLogFormatter_Format(t *testing.T) {
	clf := CommonLogFormatter{}

	testCases := []struct {
		name        string
		data        map[string]any
		expectedLog string
	}{
		{
			name: "DownstreamStatus & DownstreamContentSize are nil",
			data: map[string]any{
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       nil,
				DownstreamContentSize:  nil,
				RequestRefererHeader:   "",
				RequestUserAgentHeader: "",
				RequestCount:           0,
				RouterName:             "",
				ServiceURL:             "",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:23:00:00 +0000] "GET /foo http" - - "-" "-" 0 "-" "-" 123000ms
`,
		},
		{
			name: "all data",
			data: map[string]any{
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       123,
				DownstreamContentSize:  132,
				RequestRefererHeader:   "referer",
				RequestUserAgentHeader: "agent",
				RequestCount:           nil,
				RouterName:             "foo",
				ServiceURL:             "http://10.0.0.2/toto",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:23:00:00 +0000] "GET /foo http" 123 132 "referer" "agent" - "foo" "http://10.0.0.2/toto" 123000ms
`,
		},
		{
			name: "all data with local time",
			data: map[string]any{
				StartLocal:             time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       123,
				DownstreamContentSize:  132,
				RequestRefererHeader:   "referer",
				RequestUserAgentHeader: "agent",
				RequestCount:           nil,
				RouterName:             "foo",
				ServiceURL:             "http://10.0.0.2/toto",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:14:00:00 -0900] "GET /foo http" 123 132 "referer" "agent" - "foo" "http://10.0.0.2/toto" 123000ms
`,
		},
		{
			name: "user-agent with double quote is escaped",
			data: map[string]any{
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               1 * time.Millisecond,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "-",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/",
				RequestProtocol:        "HTTP/1.1",
				DownstreamStatus:       200,
				DownstreamContentSize:  0,
				RequestRefererHeader:   "-",
				RequestUserAgentHeader: `foo " bar`,
				RequestCount:           1,
				RouterName:             "test@file",
				ServiceURL:             "http://127.0.0.1:8080",
			},
			expectedLog: `10.0.0.1 - - [10/Nov/2009:23:00:00 +0000] "GET / HTTP/1.1" 200 0 "-" "foo \" bar" 1 "test@file" "http://127.0.0.1:8080" 1ms
`,
		},
	}

	var err error
	time.Local, err = time.LoadLocation("Etc/GMT+9")
	require.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			entry := &logrus.Entry{Data: test.data}

			raw, err := clf.Format(entry)
			assert.NoError(t, err)

			assert.Equal(t, test.expectedLog, string(raw))
		})
	}
}

func TestGenericCLFLogFormatter_Format(t *testing.T) {
	clf := GenericCLFLogFormatter{}

	testCases := []struct {
		name        string
		data        map[string]any
		expectedLog string
	}{
		{
			name: "DownstreamStatus & DownstreamContentSize are nil",
			data: map[string]any{
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       nil,
				DownstreamContentSize:  nil,
				RequestRefererHeader:   "",
				RequestUserAgentHeader: "",
				RequestCount:           0,
				RouterName:             "",
				ServiceURL:             "",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:23:00:00 +0000] "GET /foo http" - - "-" "-"
`,
		},
		{
			name: "all data",
			data: map[string]any{
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       123,
				DownstreamContentSize:  132,
				RequestRefererHeader:   "referer",
				RequestUserAgentHeader: "agent",
				RequestCount:           nil,
				RouterName:             "foo",
				ServiceURL:             "http://10.0.0.2/toto",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:23:00:00 +0000] "GET /foo http" 123 132 "referer" "agent"
`,
		},
		{
			name: "all data with local time",
			data: map[string]any{
				StartLocal:             time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:               123 * time.Second,
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "http",
				DownstreamStatus:       123,
				DownstreamContentSize:  132,
				RequestRefererHeader:   "referer",
				RequestUserAgentHeader: "agent",
				RequestCount:           nil,
				RouterName:             "foo",
				ServiceURL:             "http://10.0.0.2/toto",
			},
			expectedLog: `10.0.0.1 - Client [10/Nov/2009:14:00:00 -0900] "GET /foo http" 123 132 "referer" "agent"
`,
		},
	}

	var err error
	time.Local, err = time.LoadLocation("Etc/GMT+9")
	require.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			entry := &logrus.Entry{Data: test.data}

			raw, err := clf.Format(entry)
			assert.NoError(t, err)

			assert.Equal(t, test.expectedLog, string(raw))
		})
	}
}

func Test_toLog(t *testing.T) {
	testCases := []struct {
		desc         string
		fields       logrus.Fields
		fieldName    string
		defaultValue string
		quoted       bool
		expectedLog  any
	}{
		{
			desc: "Should return int 1",
			fields: logrus.Fields{
				"Powpow": 1,
			},
			fieldName:    "Powpow",
			defaultValue: defaultValue,
			quoted:       false,
			expectedLog:  1,
		},
		{
			desc: "Should return string foo",
			fields: logrus.Fields{
				"Powpow": "foo",
			},
			fieldName:    "Powpow",
			defaultValue: defaultValue,
			quoted:       true,
			expectedLog:  `"foo"`,
		},
		{
			desc: "Should escape double quotes in quoted string",
			fields: logrus.Fields{
				"Powpow": `foo " bar`,
			},
			fieldName:    "Powpow",
			defaultValue: defaultValue,
			quoted:       true,
			expectedLog:  `"foo \" bar"`,
		},
		{
			desc: "Should return defaultValue if fieldName does not exist",
			fields: logrus.Fields{
				"Powpow": "foo",
			},
			fieldName:    "",
			defaultValue: defaultValue,
			quoted:       false,
			expectedLog:  "-",
		},
		{
			desc:         "Should return defaultValue if fields is nil",
			fields:       nil,
			fieldName:    "",
			defaultValue: defaultValue,
			quoted:       false,
			expectedLog:  "-",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			lg := toLog(test.fields, test.fieldName, defaultValue, test.quoted)

			assert.Equal(t, test.expectedLog, lg)
		})
	}
}

func TestCustomLogFormatter_Format(t *testing.T) {
	testCases := []struct {
		desc        string
		template    string
		data        map[string]any
		expectedLog string
	}{
		{
			desc:     "all data",
			template: `{RequestHost} {ClientHost} - {ClientUsername} [{StartUTC}] "{RequestMethod} {RequestPath} {RequestProtocol}" {DownstreamStatus} {DownstreamContentSize} "{request_Referer}" "{request_User-Agent}" {Duration} {GzipRatio}`,
			data: map[string]any{
				RequestHost:            "example.com",
				ClientHost:             "10.0.0.1",
				ClientUsername:         "Client",
				StartUTC:               time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				RequestMethod:          http.MethodGet,
				RequestPath:            "/foo",
				RequestProtocol:        "HTTP/1.1",
				DownstreamStatus:       200,
				DownstreamContentSize:  int64(132),
				RequestRefererHeader:   "referer",
				RequestUserAgentHeader: "agent",
				Duration:               123 * time.Millisecond,
				GzipRatio:              2.5,
			},
			expectedLog: `example.com 10.0.0.1 - Client [2009-11-10T23:00:00Z] "GET /foo HTTP/1.1" 200 132 "referer" "agent" 123000000 2.5
`,
		},
		{
			desc:     "missing, nil and empty values",
			template: `{RequestHost} {ClientUsername} {RequestPath}`,
			data: map[string]any{
				ClientUsername: nil,
				RequestPath:    "",
			},
			expectedLog: `- - -
`,
		},
		{
			desc:     "string null is not a missing value",
			template: `{ClientUsername}`,
			data: map[string]any{
				ClientUsername: "null",
			},
			expectedLog: `null
`,
		},
		{
			desc:     "quotes, backslashes and control characters are escaped",
			template: `"{RequestPath}" "{request_User-Agent}"`,
			data: map[string]any{
				RequestPath:            `/foo?q="x&y=<1>`,
				RequestUserAgentHeader: "evil\\\" 200\x1b\n",
			},
			expectedLog: `"/foo?q=\"x&y=<1>" "evil\\\" 200\u001b\n"
`,
		},
		{
			desc:     "error values are logged with their message",
			template: `{RouterName}`,
			data: map[string]any{
				RouterName: errors.New("boom"),
			},
			expectedLog: `boom
`,
		},
		{
			desc:     "header names are canonicalized",
			template: `{request_user-agent} {origin_content-type} {downstream_X-Custom}`,
			data: map[string]any{
				"request_User-Agent":   "agent",
				"origin_Content-Type":  "text/plain",
				"downstream_X-Custom":  "REDACTED",
				"downstream_x-custom":  "not canonical",
				"origin_content-type":  "not canonical",
				"request_user-agent":   "not canonical",
				"request_Unrelated-To": "unrelated",
			},
			expectedLog: `agent text/plain REDACTED
`,
		},
		{
			desc:     "template without placeholder",
			template: `static line`,
			expectedLog: `static line
`,
		},
		{
			desc:     "clf time and ms duration modifiers",
			template: `[{StartUTC:clf}] [{StartLocal:clf}] {Duration:ms}ms {OriginDuration:ms}ms {Overhead:ms}ms`,
			data: map[string]any{
				StartUTC:       time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				StartLocal:     time.Date(2009, time.November, 10, 14, 0, 0, 0, time.FixedZone("", -9*60*60)),
				Duration:       123*time.Millisecond + 999*time.Microsecond,
				OriginDuration: 100 * time.Millisecond,
				Overhead:       500 * time.Microsecond,
			},
			expectedLog: `[10/Nov/2009:23:00:00 +0000] [10/Nov/2009:14:00:00 -0900] 123ms 100ms 0ms
`,
		},
		{
			desc:     "modifiers on missing values",
			template: `[{StartUTC:clf}] {Duration:ms}`,
			expectedLog: `[-] -
`,
		},
		{
			desc:     "escaped braces and backslashes",
			template: `\{"host": "{RequestHost}"\} \\{RequestHost} \n`,
			data: map[string]any{
				RequestHost: "example.com",
			},
			expectedLog: `{"host": "example.com"} \example.com \n
`,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			formatter, err := NewCustomLogFormatter(test.template)
			require.NoError(t, err)

			raw, err := formatter.Format(&logrus.Entry{Data: test.data})
			require.NoError(t, err)

			assert.Equal(t, test.expectedLog, string(raw))
		})
	}
}

func TestCustomLogFormatter_fields(t *testing.T) {
	formatter, err := NewCustomLogFormatter(`{Duration} {request_user-agent} {Duration:ms} {request_User-Agent} {RequestHost}`)
	require.NoError(t, err)

	assert.Equal(t, []string{Duration, RequestUserAgentHeader, RequestHost}, formatter.fields())
}

func TestNewCustomLogFormatter_invalidTemplate(t *testing.T) {
	testCases := []struct {
		desc          string
		template      string
		expectedError string
	}{
		{
			desc:          "empty template",
			template:      "",
			expectedError: "template is empty",
		},
		{
			desc:          "unknown field",
			template:      "{RequestHost} {Unknown}",
			expectedError: "unknown field in placeholder {Unknown}",
		},
		{
			desc:          "field names are case-sensitive",
			template:      "{requesthost}",
			expectedError: "unknown field in placeholder {requesthost}",
		},
		{
			desc:          "empty placeholder",
			template:      "{}",
			expectedError: "unknown field in placeholder {}",
		},
		{
			desc:          "unclosed placeholder",
			template:      "foo {RequestHost",
			expectedError: "unclosed placeholder at position 4",
		},
		{
			desc:          "nested placeholder",
			template:      "{Request{Host}",
			expectedError: "unclosed placeholder at position 0",
		},
		{
			desc:          "unexpected closing brace",
			template:      "{RequestHost} }",
			expectedError: "unexpected '}' at position 14",
		},
		{
			desc:          "empty header name",
			template:      "{request_}",
			expectedError: "invalid header name in placeholder {request_}",
		},
		{
			desc:          "invalid header name",
			template:      "{request_User Agent}",
			expectedError: "invalid header name in placeholder {request_User Agent}",
		},
		{
			desc:          "unknown modifier",
			template:      "{StartUTC:iso}",
			expectedError: "unknown modifier in placeholder {StartUTC:iso}",
		},
		{
			desc:          "empty modifier",
			template:      "{StartUTC:}",
			expectedError: "unknown modifier in placeholder {StartUTC:}",
		},
		{
			desc:          "time modifier on a non-time field",
			template:      "{RequestHost:clf}",
			expectedError: `modifier "clf" only applies to StartUTC, StartLocal, in placeholder {RequestHost:clf}`,
		},
		{
			desc:          "duration modifier on a non-duration field",
			template:      "{StartUTC:ms}",
			expectedError: `modifier "ms" only applies to Duration, OriginDuration, Overhead, in placeholder {StartUTC:ms}`,
		},
		{
			desc:          "escaped opening brace leaves the closing brace unexpected",
			template:      `\{RequestHost}`,
			expectedError: "unexpected '}' at position 13",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			_, err := NewCustomLogFormatter(test.template)
			assert.EqualError(t, err, test.expectedError)
		})
	}
}
