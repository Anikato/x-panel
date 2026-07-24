package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

const (
	sensitiveBodyOmitted = "[sensitive body omitted]"
	invalidJSONOmitted   = "[request body omitted: invalid json]"
	oversizedBodyOmitted = "[request body omitted: exceeds 1048576 bytes]"
)

var sensitiveOperationPaths = []string{
	"/api/v1/auth/password",
	"/api/v1/settings",
	"/api/v1/hosts",
	"/api/v1/nodes",
	"/api/v1/databases",
	"/api/v1/backup",
	"/api/v1/disk/remote",
	"/api/v1/acme-accounts",
	"/api/v1/dns-accounts",
	"/api/v1/certificates",
	"/api/v1/cert-sources",
	"/api/v1/cert-server",
	"/api/v1/ssl/accounts",
	"/api/v1/ssh/keys",
	"/api/v1/host/users",
	"/api/v1/toolbox/samba/users",
	"/api/v1/cronjobs",
	"/api/v1/websites",
	"/api/v1/files/content",
	"/api/v1/files/save",
	"/api/v1/commands",
	"/api/v1/nginx/conf",
	"/api/v1/nginx/conf-file",
	"/api/v1/containers",
	"/api/v1/gost/services",
	"/api/v1/gost/chains",
	"/api/v1/toolbox/services",
	"/api/v1/haproxy/config",
	"/api/v1/haproxy/stats/settings",
}

var sensitiveJSONKeys = map[string]struct{}{
	"password":         {},
	"newpassword":      {},
	"oldpassword":      {},
	"sshpassword":      {},
	"basicpassword":    {},
	"encryptpassword":  {},
	"secret":           {},
	"secretkey":        {},
	"mfasecret":        {},
	"sessionsecret":    {},
	"jwtsecret":        {},
	"clientsecret":     {},
	"token":            {},
	"agenttoken":       {},
	"githubtoken":      {},
	"privatekey":       {},
	"passphrase":       {},
	"accesskey":        {},
	"accesskeyid":      {},
	"credential":       {},
	"authorization":    {},
	"cookie":           {},
	"connectionstring": {},
	"eabhmackey":       {},
	"authpass":         {},
	"statspass":        {},
}

var bearerTokenPattern = regexp.MustCompile(
	`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`,
)

var sensitiveAssignmentPattern = regexp.MustCompile(
	`(?i)"?\b(password|new[_-]?password|old[_-]?password|token|agent[_-]?token|github[_-]?token|private[_-]?key|pass[_-]?phrase|access[_-]?key|credential|authorization|cookie|connection[_-]?string|eab[_-]?hmac[_-]?key|auth[_-]?pass|secret)\b"?\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,;]+)`,
)

func isSensitiveOperationPath(path string) bool {
	for _, prefix := range sensitiveOperationPaths {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

type replayReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *replayReadCloser) Close() error {
	return r.closer.Close()
}

func replaceRequestBody(
	request *http.Request,
	prefix []byte,
	remainder io.Reader,
	closer io.Closer,
) {
	request.Body = &replayReadCloser{
		Reader: io.MultiReader(bytes.NewReader(prefix), remainder),
		closer: closer,
	}
}

func operationLogMediaType(raw string) string {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil || mediaType == "" {
		return "unknown"
	}
	return strings.ToLower(mediaType)
}

func captureOperationLogBody(request *http.Request, path string) string {
	if request.Body == nil {
		return ""
	}
	if isSensitiveOperationPath(path) {
		return sensitiveBodyOmitted
	}

	mediaType := operationLogMediaType(request.Header.Get("Content-Type"))
	if mediaType != "application/json" && !strings.HasSuffix(mediaType, "+json") {
		return fmt.Sprintf("[request body omitted: %s]", mediaType)
	}

	original := request.Body
	prefix, err := io.ReadAll(io.LimitReader(original, maxLogBodySize+1))
	replaceRequestBody(request, prefix, original, original)
	if err != nil {
		return "[request body omitted: read error]"
	}
	if len(prefix) > maxLogBodySize {
		return oversizedBodyOmitted
	}
	return sanitizeOperationJSON(prefix)
}

func sanitizeOperationJSON(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}

	var value interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return invalidJSONOmitted
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return invalidJSONOmitted
	}

	redactOperationJSON(value)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return invalidJSONOmitted
	}
	return string(sanitized)
}

func redactOperationJSON(value interface{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if isSensitiveJSONKey(key) {
				typed[key] = "***"
				continue
			}
			redactOperationJSON(item)
		}
	case []interface{}:
		for _, item := range typed {
			redactOperationJSON(item)
		}
	}
}

func isSensitiveJSONKey(key string) bool {
	normalized := strings.Map(func(char rune) rune {
		switch char {
		case '_', '-', ' ', '.':
			return -1
		default:
			return unicode.ToLower(char)
		}
	}, key)
	if _, ok := sensitiveJSONKeys[normalized]; ok {
		return true
	}
	for _, suffix := range []string{
		"password",
		"secret",
		"token",
		"privatekey",
		"passphrase",
		"credential",
		"accesskey",
		"hmackey",
	} {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}

func sanitizeOperationMessage(message string) string {
	message = bearerTokenPattern.ReplaceAllString(message, "Bearer ***")
	return sensitiveAssignmentPattern.ReplaceAllString(message, "${1}=***")
}
