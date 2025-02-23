//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import "github.com/fogfish/faults"

// Common agents errors
const (
	ErrLLM         = faults.Type("LLM I/O has failed")
	ErrUnknown     = faults.Type("unkown agent statet")
	ErrAborted     = faults.Type("execution aborted")
	ErrMaxEpoch    = faults.Safe1[int]("max epoch %d is reached")
	ErrCmdConflict = faults.Type("command already exists")
	ErrCmdInvalid  = faults.Type("invalid command specification, missing required attributes")
)
