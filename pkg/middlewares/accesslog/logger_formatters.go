package accesslog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpguts"
)

// default format for time presentation.
const (
	commonLogTimeFormat = "02/Jan/2006:15:04:05 -0700"
	defaultValue        = "-"
)

// CommonLogFormatter provides formatting in the Traefik common log format.
type CommonLogFormatter struct{}

// Format formats the log entry in the Traefik common log format.
func (f *CommonLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	timestamp := defaultValue
	if v, ok := entry.Data[StartUTC]; ok {
		timestamp = v.(time.Time).Format(commonLogTimeFormat)
	} else if v, ok := entry.Data[StartLocal]; ok {
		timestamp = v.(time.Time).Local().Format(commonLogTimeFormat)
	}

	var elapsedMillis int64
	if v, ok := entry.Data[Duration]; ok {
		elapsedMillis = v.(time.Duration).Nanoseconds() / 1000000
	}

	_, err := fmt.Fprintf(b, "%s - %s [%s] \"%s %s %s\" %v %v %s %s %v %s %s %dms\n",
		toLog(entry.Data, ClientHost, defaultValue, false),
		toLog(entry.Data, ClientUsername, defaultValue, false),
		timestamp,
		toLog(entry.Data, RequestMethod, defaultValue, false),
		toLog(entry.Data, RequestPath, defaultValue, false),
		toLog(entry.Data, RequestProtocol, defaultValue, false),
		toLog(entry.Data, DownstreamStatus, defaultValue, true),
		toLog(entry.Data, DownstreamContentSize, defaultValue, true),
		toLog(entry.Data, "request_Referer", `"-"`, true),
		toLog(entry.Data, "request_User-Agent", `"-"`, true),
		toLog(entry.Data, RequestCount, defaultValue, true),
		toLog(entry.Data, RouterName, `"-"`, true),
		toLog(entry.Data, ServiceURL, `"-"`, true),
		elapsedMillis)

	return b.Bytes(), err
}

// GenericCLFLogFormatter provides formatting in the generic CLF log format.
type GenericCLFLogFormatter struct{}

// Format formats the log entry in the generic CLF log format.
func (f *GenericCLFLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	timestamp := defaultValue
	if v, ok := entry.Data[StartUTC]; ok {
		timestamp = v.(time.Time).Format(commonLogTimeFormat)
	} else if v, ok := entry.Data[StartLocal]; ok {
		timestamp = v.(time.Time).Local().Format(commonLogTimeFormat)
	}

	_, err := fmt.Fprintf(b, "%s - %s [%s] \"%s %s %s\" %v %v %s %s\n",
		toLog(entry.Data, ClientHost, defaultValue, false),
		toLog(entry.Data, ClientUsername, defaultValue, false),
		timestamp,
		toLog(entry.Data, RequestMethod, defaultValue, false),
		toLog(entry.Data, RequestPath, defaultValue, false),
		toLog(entry.Data, RequestProtocol, defaultValue, false),
		toLog(entry.Data, DownstreamStatus, defaultValue, true),
		toLog(entry.Data, DownstreamContentSize, defaultValue, true),
		toLog(entry.Data, "request_Referer", `"-"`, true),
		toLog(entry.Data, "request_User-Agent", `"-"`, true))

	return b.Bytes(), err
}

// CustomLogFormatter provides formatting with a user-defined template.
type CustomLogFormatter struct {
	segments []templateSegment
}

// Template modifiers, written as {FieldName:modifier}, change how a field value is written.
const (
	// clfTimeModifier writes a time with the Common Log Format layout, as the common format does.
	clfTimeModifier = "clf"
	// msDurationModifier writes a duration in milliseconds, as the common format does.
	msDurationModifier = "ms"
)

// modifierFields lists the fields each modifier applies to, so that a modifier on another field is reported at startup.
var modifierFields = map[string][]string{
	clfTimeModifier:    {StartUTC, StartLocal},
	msDurationModifier: {Duration, OriginDuration, Overhead},
}

// templateSegment is either a literal text, or a placeholder for the field it names.
type templateSegment struct {
	literal  string
	field    string
	modifier string
}

// NewCustomLogFormatter creates a CustomLogFormatter from a template where each {FieldName} placeholder
// is replaced by the value of the field, as logged by the json format.
// A backslash escapes a brace or another backslash, so that they are written as is.
// The template is parsed once, so that invalid placeholders are reported at startup instead of on every log line.
func NewCustomLogFormatter(template string) (*CustomLogFormatter, error) {
	if template == "" {
		return nil, errors.New("template is empty")
	}

	var segments []templateSegment
	var literal strings.Builder

	for i := 0; i < len(template); i++ {
		switch template[i] {
		case '\\':
			if i+1 < len(template) && strings.IndexByte(`\{}`, template[i+1]) >= 0 {
				i++
			}

			literal.WriteByte(template[i])

		case '}':
			return nil, fmt.Errorf("unexpected '}' at position %d", i)

		case '{':
			length := strings.IndexAny(template[i+1:], "{}")
			if length < 0 || template[i+1+length] == '{' {
				return nil, fmt.Errorf("unclosed placeholder at position %d", i)
			}

			segment, err := parsePlaceholder(template[i+1 : i+1+length])
			if err != nil {
				return nil, err
			}

			if literal.Len() > 0 {
				segments = append(segments, templateSegment{literal: literal.String()})
				literal.Reset()
			}

			segments = append(segments, segment)
			i += length + 1

		default:
			literal.WriteByte(template[i])
		}
	}

	if literal.Len() > 0 {
		segments = append(segments, templateSegment{literal: literal.String()})
	}

	return &CustomLogFormatter{segments: segments}, nil
}

// Format formats the log entry with the template.
func (f *CustomLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	value := &bytes.Buffer{}
	encoder := json.NewEncoder(value)
	// Unlike the json format, HTML characters are not escaped, as they are common in URLs and not special in a log line.
	encoder.SetEscapeHTML(false)

	for _, segment := range f.segments {
		if segment.field == "" {
			b.WriteString(segment.literal)
			continue
		}

		if writeModifiedValue(b, segment.modifier, entry.Data[segment.field]) {
			continue
		}

		if err := writeTemplateValue(b, value, encoder, entry.Data[segment.field]); err != nil {
			return nil, fmt.Errorf("formatting field %s: %w", segment.field, err)
		}
	}

	b.WriteByte('\n')

	return b.Bytes(), nil
}

// fields returns the fields used by the template, each once, in template order.
func (f *CustomLogFormatter) fields() []string {
	var fields []string
	for _, segment := range f.segments {
		if segment.field != "" && !slices.Contains(fields, segment.field) {
			fields = append(fields, segment.field)
		}
	}

	return fields
}

// parsePlaceholder parses the content of a placeholder, written as FieldName or FieldName:modifier.
func parsePlaceholder(content string) (templateSegment, error) {
	name, modifier, hasModifier := strings.Cut(content, ":")

	field, err := resolveTemplateField(name)
	if err != nil {
		return templateSegment{}, err
	}

	if hasModifier {
		fields, ok := modifierFields[modifier]
		if !ok {
			return templateSegment{}, fmt.Errorf("unknown modifier in placeholder {%s}", content)
		}

		if !slices.Contains(fields, field) {
			return templateSegment{}, fmt.Errorf("modifier %q only applies to %s, in placeholder {%s}", modifier, strings.Join(fields, ", "), content)
		}
	}

	return templateSegment{field: field, modifier: modifier}, nil
}

// writeModifiedValue writes the value with the modifier, and reports whether it did.
// Values of an unexpected type, such as missing values, are left to the default rendering.
func writeModifiedValue(b *bytes.Buffer, modifier string, value any) bool {
	switch modifier {
	case clfTimeModifier:
		if t, ok := value.(time.Time); ok {
			b.WriteString(t.Format(commonLogTimeFormat))
			return true
		}

	case msDurationModifier:
		if d, ok := value.(time.Duration); ok {
			b.WriteString(strconv.FormatInt(d.Milliseconds(), 10))
			return true
		}
	}

	return false
}

// resolveTemplateField returns the log entry key for a placeholder name.
// Names match the json format keys, and the header part of header fields is canonicalized like header names in the fields configuration.
func resolveTemplateField(name string) (string, error) {
	if prefix, header, found := cutHeaderPrefix(name); found {
		if !httpguts.ValidHeaderFieldName(header) {
			return "", fmt.Errorf("invalid header name in placeholder {%s}", name)
		}

		return prefix + textproto.CanonicalMIMEHeaderKey(header), nil
	}

	if _, ok := allCoreKeys[name]; !ok {
		return "", fmt.Errorf("unknown field in placeholder {%s}", name)
	}

	return name, nil
}

func cutHeaderPrefix(field string) (prefix, header string, found bool) {
	for _, prefix := range []string{RequestHeaderPrefix, OriginHeaderPrefix, DownstreamHeaderPrefix} {
		if header, found := strings.CutPrefix(field, prefix); found {
			return prefix, header, true
		}
	}

	return "", "", false
}

// writeTemplateValue writes the value as encoded by the json format, so that both formats log the same values,
// and the json escaping keeps quotes and control characters from breaking the line.
func writeTemplateValue(b, scratch *bytes.Buffer, encoder *json.Encoder, value any) error {
	// Errors are logged with their message, like logrus.JSONFormatter does.
	if err, ok := value.(error); ok {
		value = err.Error()
	}

	scratch.Reset()
	if err := encoder.Encode(value); err != nil {
		return err
	}

	raw := bytes.TrimSuffix(scratch.Bytes(), []byte("\n"))
	if string(raw) == "null" {
		b.WriteString(defaultValue)
		return nil
	}

	// Strings are written without their surrounding quotes, so that the template decides where quotes go.
	if raw[0] == '"' {
		raw = raw[1 : len(raw)-1]
	}

	if len(raw) == 0 {
		b.WriteString(defaultValue)
		return nil
	}

	b.Write(raw)

	return nil
}

func toLog(fields logrus.Fields, key, defaultValue string, quoted bool) any {
	if v, ok := fields[key]; ok {
		if v == nil {
			return defaultValue
		}

		switch s := v.(type) {
		case string:
			return toLogEntry(s, defaultValue, quoted)

		case fmt.Stringer:
			return toLogEntry(s.String(), defaultValue, quoted)

		default:
			return v
		}
	}

	return defaultValue
}

func toLogEntry(s, defaultValue string, quote bool) string {
	if len(s) == 0 {
		return defaultValue
	}

	if quote {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
