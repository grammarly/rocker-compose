/*-
 * Copyright 2015 Grammarly, Inc.
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

package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
)

// GenerateRandomID returns an unique id
func GenerateRandomID() string {
	for {
		id := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, id); err != nil {
			panic(err) // This shouldn't happen
		}
		value := hex.EncodeToString(id)
		// if we try to parse the truncated for as an int and we don't have
		// an error then the value is all numberic and causes issues when
		// used as a hostname. ref #3869
		if _, err := strconv.ParseInt(fmt.Sprintf("%.12s", value), 10, 64); err == nil {
			continue
		}
		return value
	}
}
