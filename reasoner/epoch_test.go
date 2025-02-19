//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package reasoner_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/reasoner"
)

func TestEpoch(t *testing.T) {
	r := reasoner.NewEpoch(5, reasoner.NewVoid[string, string]())

	t.Run("Pass", func(t *testing.T) {
		s := thinker.State[string, string]{Epoch: 0}
		_, _, err := r.Deduct(s)

		it.Then(t).Should(
			it.Nil(err),
		)
	})

	t.Run("Fail", func(t *testing.T) {
		s := thinker.State[string, string]{Epoch: 5}
		_, _, err := r.Deduct(s)

		it.Then(t).Should(
			it.String(err.Error()).Contain("max epoch"),
		)
	})
}
