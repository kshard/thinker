//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
)

func TestNewSeqRegistry(t *testing.T) {
	seq := command.NewSeqRegistry()

	it.Then(t).Should(
		it.True(seq != nil),
	)
}

func TestSeqRegistryBind(t *testing.T) {
	t.Run("BindSingle", func(t *testing.T) {
		seq := command.NewSeqRegistry()
		reg := command.NewRegistry()
		reg.Attach("fs", mockOne("read", "Read file"))

		seq.Bind(reg)

		ctx := seq.Context()
		it.Then(t).Should(
			it.Equal(len(ctx), 1),
		)
	})

	t.Run("BindMultiple", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg1 := command.NewRegistry()
		reg1.Attach("fs", mockOne("read", "Read file"))

		reg2 := command.NewRegistry()
		reg2.Attach("db", mockOne("query", "Query database"))

		seq.Bind(reg1)
		seq.Bind(reg2)

		ctx := seq.Context()
		it.Then(t).Should(
			it.Equal(len(ctx), 2),
		)
	})

	t.Run("BindEmpty", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		ctx := seq.Context()
		it.Then(t).Should(
			it.Equal(len(ctx), 0),
		)
	})
}

func TestSeqRegistryContext(t *testing.T) {
	t.Run("CombinesToolsFromAllRegistries", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg1 := command.NewRegistry()
		reg1.Attach("fs", mockSeq(2, "read", "Read file"))

		reg2 := command.NewRegistry()
		reg2.Attach("db", mockSeq(3, "query", "Query database"))

		seq.Bind(reg1)
		seq.Bind(reg2)

		ctx := seq.Context()
		it.Then(t).Should(
			it.Equal(len(ctx), 5),
		)
	})

	t.Run("ContextIsCached", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg := command.NewRegistry()
		reg.Attach("fs", mockOne("read", "Read file"))

		seq.Bind(reg)

		ctx1 := seq.Context()
		ctx2 := seq.Context()

		it.Then(t).Should(
			it.Equal(len(ctx1), len(ctx2)),
		)
	})
}

func TestSeqRegistryInvoke(t *testing.T) {
	t.Run("InvokePrefixedTool", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg := command.NewRegistry()
		reg.Attach("fs", mockReply("read", "Read file", "file contents"))
		seq.Bind(reg)
		seq.Context()

		reply := replyOne("fs_read", map[string]any{"path": "/test.txt"})
		phase, msg, err := seq.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
			it.True(msg != nil),
		)
	})

	t.Run("InvokeToolFromSecondRegistry", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg1 := command.NewRegistry()
		reg1.Attach("fs", mockReply("read", "Read file", "file contents"))

		reg2 := command.NewRegistry()
		reg2.Attach("db", mockReply("query", "Query database", "query results"))

		seq.Bind(reg1)
		seq.Bind(reg2)
		seq.Context()

		reply := replyOne("db_query", map[string]any{"sql": "SELECT 1"})
		phase, msg, err := seq.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
			it.True(msg != nil),
		)
	})

	t.Run("InvokeUnprefixedTool", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg := command.NewRegistry()
		reg.Attach("fs", mockReply("read", "Read file", "file contents"))
		seq.Bind(reg)
		seq.Context()

		reply := replyOne("tool", map[string]any{})
		phase, _, err := seq.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
		)
	})

	t.Run("InvokeUnknownPrefix", func(t *testing.T) {
		seq := command.NewSeqRegistry()

		reg := command.NewRegistry()
		reg.Attach("fs", mockReply("read", "Read file", "file contents"))
		seq.Bind(reg)
		seq.Context()

		reply := replyOne("unknown_tool", map[string]any{})
		phase, _, err := seq.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
		)
	})
}
