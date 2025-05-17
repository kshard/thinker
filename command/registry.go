//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

type Registry struct {
	mu       sync.Mutex
	registry map[string]thinker.Cmd
	cmds     chatter.Registry
}

var _ thinker.Registry = (*Registry)(nil)

func NewRegistry() *Registry {
	r := &Registry{
		registry: make(map[string]thinker.Cmd),
		cmds:     chatter.Registry{},
	}
	r.Register(Return())
	return r
}

// Register new command
func (r *Registry) Register(cmd thinker.Cmd) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, has := r.registry[cmd.Cmd]; has {
		return thinker.ErrCmdConflict
	}

	if !cmd.IsValid() {
		return thinker.ErrCmdInvalid
	}

	c, err := convert(cmd)
	if err != nil {
		return thinker.ErrCmdInvalid
	}

	r.registry[cmd.Cmd] = cmd
	r.cmds = append(r.cmds, c)
	return nil
}

func (r *Registry) Context() chatter.Registry { return r.cmds }

func (r *Registry) Invoke(reply *chatter.Reply) (thinker.Phase, chatter.Message, error) {
	var hasReturn = false

	answer, err := reply.Invoke(func(name string, args json.RawMessage) (json.RawMessage, error) {
		cmd, has := r.registry[name]
		if !has {
			return nil, thinker.Feedback(
				fmt.Sprintf("the tool %s is unknown to the client.", name),
			)
		}

		b, err := cmd.Run(args)
		if err != nil {
			var feedback chatter.Content
			if ok := errors.As(err, &feedback); !ok {
				return nil, err
			}

			exx := feedback.String()
			return pack([]byte(exx))
		}

		if name == RETURN {
			hasReturn = true
			return b, nil
		}

		return pack(b)
	})

	if err != nil {
		return thinker.AGENT_ABORT, nil, err
	}

	if hasReturn {
		for _, yield := range answer.Yield {
			if yield.Source == RETURN {
				return thinker.AGENT_RETURN, chatter.Text(yield.Value), nil
			}
		}

		return thinker.AGENT_RETURN, nil, nil
	}

	return thinker.AGENT_ASK, answer, nil
}

func convert(cmd thinker.Cmd) (chatter.Cmd, error) {
	required := make([]string, 0)
	properties := map[string]any{}
	for _, arg := range cmd.Args {
		properties[arg.Name] = arg
		required = append(required, arg.Name)
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}

	raw, err := json.Marshal(schema)
	if err != nil {
		return chatter.Cmd{}, err
	}

	return chatter.Cmd{
		Cmd:    cmd.Cmd,
		About:  cmd.About,
		Schema: raw,
	}, nil
}

func pack(b []byte) (json.RawMessage, error) {
	pckt := map[string]any{
		"toolOutput": string(b),
	}

	bin, err := json.Marshal(pckt)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bin), nil

}
