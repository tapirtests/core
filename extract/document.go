package extract

import (
	"encoding/json"
	"strconv"

	"github.com/tapirtests/core/protocol"
)

// Parts of the response document: the first name of every path.
const (
	// PartStatus is the status of the response, a number: $.status.
	PartStatus = "status"
	// PartHeaders are the response headers, an object of strings with
	// lower-case names: $.headers['content-type'].
	PartHeaders = "headers"
	// PartBody is the parsed response body: $.body.items[0].name.
	PartBody = "body"
	// PartDuration is how long the call took, a number of milliseconds:
	// $.duration.
	PartDuration = "duration"
)

// Parts lists the parts of the response document.
var Parts = []string{PartStatus, PartHeaders, PartBody, PartDuration}

// Document builds the response document that paths are applied to:
//
//	{
//	  "status":   201,
//	  "headers":  {"content-type": "application/json"},
//	  "body":     {"id": 57, "name": "Phone"},
//	  "duration": 84
//	}
//
// Both extraction and assertions use this one document, so a path means the
// same thing everywhere. Numbers are json.Number, like all numbers decoded
// from JSON.
func Document(res *protocol.Result) map[string]any {
	headers := make(map[string]any, len(res.Headers))
	for name, value := range res.Headers {
		headers[name] = value
	}
	return map[string]any{
		PartStatus:   json.Number(strconv.Itoa(res.Status)),
		PartHeaders:  headers,
		PartBody:     res.Body,
		PartDuration: json.Number(strconv.FormatInt(res.Duration.Milliseconds(), 10)),
	}
}
