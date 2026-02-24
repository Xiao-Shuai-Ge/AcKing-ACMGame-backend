package types

import "encoding/json"

type WsRequest struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type WsResponse struct {
	Type    string      `json:"type"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
