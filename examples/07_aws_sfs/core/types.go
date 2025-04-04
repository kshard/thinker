//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package core

// Raw ingested document
type Document struct {
	Text string `json:"text"`
}

// Summary of the document
type Abstract struct {
	Text string `json:"text"`
}

// Classified version of document
type Keywords struct {
	Keywords string `json:"type"`
	Text     string `json:"text"`
}

// Insight
type Insight struct {
	Concern string `json:"concern"`
}
