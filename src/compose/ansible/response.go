package ansible

import (
	"encoding/json"
	"io"
)

type Response struct {
	Changed bool                `json:"changed"`
	Failed  bool                `json:"failed"`
	Message string              `json:"msg"`
	Removed []ResponseContainer `json:"removed"`
	Created []ResponseContainer `json:"created"`
	Pulled  []string            `json:"pulled"`
}

type ResponseContainer struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (r *Response) Error(msg error) *Response {
	r.Message = msg.Error()
	r.Failed = true
	return r
}

func (r *Response) Success(msg string) *Response {
	r.Message = msg
	return r
}

func (r *Response) Encode() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Response) WriteTo(w io.Writer) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return w.Write(data)
}
