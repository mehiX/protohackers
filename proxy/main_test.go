package main

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReplaceCoinAddress(t *testing.T) {

	type scenario struct {
		in  string
		out string
	}

	scenarios := []scenario{
		{
			in:  "Please pay the ticket price of 15 Boguscoins to one of these addresses: 7KQxlui6oOsFuQ6aogRAaVID14kemIGjB74 7bXsizxWneweRdeg5nNQ2Aasdqweqeqewdkjas8weruiwksadjfx 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7KQxlui6oOsFuQ6aogRAaVID14kemIGjB74 7KQxlui6oOsFuQ6aogRAaVID14kemIGjB74 mC25I7YWHMfk9JZe0LM0g1ZauHuiSxhI 7W3WEulptLbpsegZuzlawzNaD9IkPpqmqK-5LdIGPWcyQqkTjNYqSfxw5FtyTSKbR-1234\n",
			out: "Please pay the ticket price of 15 Boguscoins to one of these addresses: 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7bXsizxWneweRdeg5nNQ2Aasdqweqeqewdkjas8weruiwksadjfx 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7YWHMfk9JZe0LM0g1ZauHuiSxhI mC25I7YWHMfk9JZe0LM0g1ZauHuiSxhI 7W3WEulptLbpsegZuzlawzNaD9IkPpqmqK-5LdIGPWcyQqkTjNYqSfxw5FtyTSKbR-1234\n"},
		{
			in:  "7YWHMfk9JZe0LM0g1ZauHuiSxhI 7iwJgE2EBG5bfltBBparbakyV4\n",
			out: "7YWHMfk9JZe0LM0g1ZauHuiSxhI 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n",
		},
		{
			in:  "[SaneDev128] 7dmINvYrOw7bQdstJBzHA6E7wxFhCl 7DhihTl5ZuWHez8RdLK7sdqTR6BH5UlY17D\n",
			out: "[SaneDev128] 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n",
		},
		{
			in:  "[SaneCaster363] 7L573mbh6CcnaPlvmVcvMkzrCEd7JwJqaOl 7EgNEEqRXjH1lLrRISslA1Ff8hIQ6\n",
			out: "[SaneCaster363] 7YWHMfk9JZe0LM0g1ZauHuiSxhI 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n",
		},
	}

	for _, s := range scenarios {
		t.Run(s.in, func(t *testing.T) {
			s := s
			t.Parallel()
			src := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader([]byte(s.in))), nil)
			var dst bytes.Buffer
			hijack(&dst, src)
			if dst.String() != s.out {
				t.Fatalf("wrong output.\nexpected:\n%s\ngot:\n%s", s.out, dst.String())
			}
		})
	}
}
