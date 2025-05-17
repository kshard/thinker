//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package core

import (
	"context"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

type Ingestor struct {
	*agent.Automata[Document, Abstract]
}

func NewIngestor(llm chatter.Chatter) *Ingestor {
	a := &Ingestor{}
	a.Automata = agent.NewAutomata(llm,
		memory.NewVoid(""),
		codec.FromEncoder(a.encode),
		codec.FromDecoder(a.decode),
		reasoner.NewVoid[Abstract](),
	)
	return a
}

func (lib *Ingestor) Ingest(doc Document) (Abstract, error) {
	return lib.Prompt(context.Background(), doc)
}

func (lib *Ingestor) encode(doc Document) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		You are an expert summarizer. Given the following document, read it carefully
		and understand its context, key messages, and tone. Then, generate a concise
		and coherent abstractive summary that captures the essential ideas in your
		own words. Avoid copying sentences verbatim. Focus on what the document is
		trying to convey overall, and express it clearly for someone who hasn’t read
		the original.
	`)

	prompt.WithInput("DOCUMENT:", doc.Text)

	return &prompt, nil
}

func (lib *Ingestor) decode(reply *chatter.Reply) (float64, Abstract, error) {
	return 1.0, Abstract{Text: reply.String()}, nil
}

//------------------------------------------------------------------------------

type Classifier struct {
	*agent.Automata[Abstract, Keywords]
	text string
}

func NewClassifier(llm chatter.Chatter) *Classifier {
	a := &Classifier{}
	a.Automata = agent.NewAutomata(llm,
		memory.NewVoid(""),
		codec.FromEncoder(a.encode),
		codec.FromDecoder(a.decode),
		reasoner.NewVoid[Keywords](),
	)
	return a
}

func (lib *Classifier) Classify(doc Abstract) (Keywords, error) {
	lib.text = doc.Text
	return lib.Prompt(context.Background(), doc)
}

func (lib *Classifier) encode(doc Abstract) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		You are an expert in text analysis. Given the following abstractive summary,
		analyze its content and identify the most relevant keywords that represent
		its core topics, themes, and main ideas. Extract these keywords in a concise
		list, ensuring they capture the essence of the summary without redundancy.
		Focus on nouns, key phrases, and significant terms related to
		the subject matter.
	`)

	prompt.WithInput("DOCUMENT:", doc.Text)

	return &prompt, nil
}

func (lib *Classifier) decode(reply *chatter.Reply) (float64, Keywords, error) {
	return 1.0, Keywords{Keywords: reply.String(), Text: lib.text}, nil
}

//------------------------------------------------------------------------------

type Insighter struct {
	*agent.Automata[Keywords, Insight]
}

func NewInsighter(llm chatter.Chatter) *Insighter {
	a := &Insighter{}
	a.Automata = agent.NewAutomata(llm,
		memory.NewStream(100, `
			You are an intelligent assistant with memory. You are using and remember
			context from earlier conversation history to execute the task.
		`),
		codec.FromEncoder(a.encode),
		codec.FromDecoder(a.decode),
		reasoner.NewVoid[Insight](),
	)
	return a
}

func (lib *Insighter) Insight(doc Keywords) (Insight, error) {
	return lib.Prompt(context.Background(), doc)
}

func (lib *Insighter) encode(doc Keywords) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		You receive a stream of abstractive summaries and their corresponding keyword
		lists over time. Your task is to store, track, and analyze this evolving
		information to answer high-level questions about the documents.
	`)

	prompt.WithGuide(`
			Based on the full memory of all previously seen summaries and keywords,
			provide thoughtful, concise answers to the following:`,
		`(1) What is the general concern or overarching theme reflected in these
			summaries so far?`,
		`(2) What keywords or topics are trending across documents (i.e., appearing
			frequently or growing in relevance)?`,
	)

	prompt.WithBlob("KEYWORDS:", doc.Keywords)

	prompt.WithBlob("DOCUMENT:", doc.Text)

	return &prompt, nil
}

func (lib *Insighter) decode(reply *chatter.Reply) (float64, Insight, error) {
	return 1.0, Insight{Concern: reply.String()}, nil
}
