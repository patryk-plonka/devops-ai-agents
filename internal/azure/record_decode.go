package azure

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcehealth/armresourcehealth"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

// collectionTransport validates the request with ReadTransport first, then
// isolates schema-invalid records before the SDK decodes the page. It runs below
// the SDK's payload cache, so retries and native continuation handling stay intact.
// Rejected entries become null. Independently decoded attribution travels with
// the same page outside the SDK payload and is validated before checkpointing.
type collectionTransport struct{ ReadTransport }

type pageRejectionsKey struct{}

var monitorFieldSeparators = strings.NewReplacer("-", "", "_", "")

func (t collectionTransport) Do(req *http.Request) (*http.Response, error) {
	response, err := t.ReadTransport.Do(req)
	if err != nil || response.StatusCode != http.StatusOK {
		return response, err
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
		return nil, io.ErrUnexpectedEOF
	}
	filtered, rejected, err := filterRecords(body, t.Route)
	if err != nil {
		return nil, err
	}
	if scopes, ok := req.Context().Value(pageRejectionsKey{}).(*[]health.RejectedRecordInput); ok {
		*scopes = rejected
	}
	response.Body = io.NopCloser(bytes.NewReader(filtered))
	response.ContentLength = int64(len(filtered))
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	response.Header.Set("Content-Length", strconv.Itoa(len(filtered)))
	return response, nil
}

func filterRecords(body []byte, route string) ([]byte, []health.RejectedRecordInput, error) {
	invalid := func() ([]byte, []health.RejectedRecordInput, error) {
		return nil, nil, fail("invalid_response", "Source returned an invalid collection page.", 1)
	}
	envelope := decodeCollectionEnvelope(body)
	if envelope == nil {
		return invalid()
	}
	var records []json.RawMessage
	raw, exists := envelope["value"]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &records) != nil {
		return invalid()
	}
	if next, exists := envelope["nextLink"]; exists {
		var link *string
		if json.Unmarshal(next, &link) != nil {
			return invalid()
		}
	}
	var rejected []health.RejectedRecordInput
	for i, raw := range records {
		if jsontext.Value(raw).IsValid() && validCollectionRecord(raw, route) {
			continue
		}
		records[i] = json.RawMessage("null")
		if scope := rejectedRecordScope(raw, route); scope.ID != "" {
			rejected = append(rejected, scope)
		}
	}
	// RawMessage contains valid JSON, so marshaling these two containers cannot
	// fail. Preserve the envelope/nextLink and valid record values.
	envelope["value"], _ = json.Marshal(records)
	filtered, _ := json.Marshal(envelope)
	return filtered, rejected, nil
}

func validCollectionRecord(raw json.RawMessage, route string) bool {
	switch route {
	case inventoryRoute:
		var resource armresources.GenericResourceExpanded
		return json.Unmarshal(raw, &resource) == nil
	case serviceHealthRoute:
		var event armresourcehealth.Event
		return json.Unmarshal(raw, &event) == nil
	case monitorRoute:
		var alert monitorAlert
		// The Monitor pager uses case-insensitive field matching.
		return jsonv2.Unmarshal(raw, &alert, jsonv2.MatchCaseInsensitiveNames(true)) == nil
	case healthRoute:
		var status armresourcehealth.AvailabilityStatus
		return json.Unmarshal(raw, &status) == nil
	case advisorRoute:
		var recommendation advisorRecommendation
		return json.Unmarshal(raw, &recommendation) == nil
	default:
		return false
	}
}

func rejectedRecordScope(raw json.RawMessage, route string) health.RejectedRecordInput {
	var hint health.RejectedRecordInput
	switch route {
	case inventoryRoute, healthRoute:
		hint.ID = recordString(recordField(raw, "id", false))
	case monitorRoute, advisorRoute:
		monitor := route == monitorRoute
		properties := recordField(raw, "properties", monitor)
		if monitor {
			hint.TargetResourceID = recordString(recordField(recordField(properties, "essentials", true), "targetResource", true))
		} else {
			hint.TargetResourceID = recordString(recordField(recordField(properties, "resourceMetadata", false), "resourceId", false))
		}
		if hint.TargetResourceID != "" {
			hint.ID = recordString(recordField(raw, "id", monitor))
		}
	}
	return hint
}

// Check every parent on the attribution path for duplicate fields. An
// unrelated duplicate can discard the row without making that path ambiguous.
func recordField(raw json.RawMessage, name string, caseInsensitive bool) json.RawMessage {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if start, err := decoder.Token(); err != nil || start != json.Delim('{') {
		return nil
	}
	var result json.RawMessage
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil
		}
		var field json.RawMessage
		if decoder.Decode(&field) != nil {
			return nil
		}
		// Also detect aliases recognized by the Monitor record's JSON v2
		// validator. Keep only spellings the actual pager would decode.
		matches := strings.EqualFold(key, name)
		if matches || caseInsensitive && strings.EqualFold(monitorFieldSeparators.Replace(key), name) {
			if result != nil || !matches || !caseInsensitive && key != name {
				return nil
			}
			result = field
		}
	}
	return result
}

func recordString(raw json.RawMessage) string {
	var value string
	if !jsontext.Value(raw).IsValid() || json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

// Keep exact wire keys so the pager sees the same fields as the filter.
// Reading the object members in order lets us detect duplicates before a
// later field can overwrite an earlier value.
func decodeCollectionEnvelope(body []byte) map[string]json.RawMessage {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if start, err := decoder.Token(); err != nil || start != json.Delim('{') {
		return nil
	}
	envelope := make(map[string]json.RawMessage, 2)
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil
		}
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return nil
		}
		if key != "value" && key != "nextLink" {
			// An alternate spelling can hide records or make a paged response
			// look terminal after filtering.
			if strings.EqualFold(key, "value") || strings.EqualFold(key, "nextLink") {
				return nil
			}
			continue
		}
		if _, exists := envelope[key]; exists {
			return nil
		}
		envelope[key] = raw
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
		return nil
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil
	}
	return envelope
}
