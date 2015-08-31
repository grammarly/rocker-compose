/*-
 * Copyright 2014 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	Cleaned []string            `json:"cleaned"`
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
