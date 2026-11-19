package azure

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcehealth/armresourcehealth"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4"
)

// collectionTransport validates the request with ReadTransport first, then
// isolates schema-invalid records before the SDK decodes the page. It runs below
// the SDK's payload cache, so retries and native continuation handling stay intact.
// Rejected entries become null: all adapters turn null entries into
// rejected-record gaps during normalization, without retaining their payloads.
type collectionTransport struct{ ReadTransport }

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
	filtered, err := filterRecords(body, t.Route)
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(filtered))
	response.ContentLength = int64(len(filtered))
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	response.Header.Set("Content-Length", strconv.Itoa(len(filtered)))
	return response, nil
}

func filterRecords(body []byte, route string) ([]byte, error) {
	invalid := func() ([]byte, error) {
		return nil, fail("invalid_response", "Source returned an invalid collection page.", 1)
	}
	var envelope map[string]json.RawMessage
	if route == advisorRoute {
		envelope = decodeAdvisorEnvelope(body)
	} else if json.Unmarshal(body, &envelope) != nil {
		return invalid()
	}
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
	changed := false
	for i, raw := range records {
		var err error
		switch route {
		case inventoryRoute:
			var resource armresources.GenericResourceExpanded
			err = json.Unmarshal(raw, &resource)
		case serviceHealthRoute:
			var event armresourcehealth.Event
			err = json.Unmarshal(raw, &event)
		case monitorRoute:
			var alert monitorAlert
			err = json.Unmarshal(raw, &alert)
		case healthRoute:
			var status armresourcehealth.AvailabilityStatus
			err = json.Unmarshal(raw, &status)
		case advisorRoute:
			var recommendation advisorRecommendation
			err = json.Unmarshal(raw, &recommendation)
		default:
			return invalid()
		}
		if err != nil {
			records[i] = json.RawMessage("null")
			changed = true
		}
	}
	if !changed && route != advisorRoute {
		return body, nil
	}
	// RawMessage contains valid JSON, so marshaling these two containers cannot
	// fail. Preserve the envelope/nextLink and valid record values.
	envelope["value"], _ = json.Marshal(records)
	filtered, _ := json.Marshal(envelope)
	return filtered, nil
}

// Keep exact wire keys so the pager sees the same fields as the filter.
// Reading the object members in order lets us detect duplicates before a
// later field can overwrite an earlier value.
func decodeAdvisorEnvelope(body []byte) map[string]json.RawMessage {
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
