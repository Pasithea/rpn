package rpn

import (
	"math/big"
	"testing"
)

var testCase = []struct {
	in      string
	postfix []string
	result  *big.Rat
	canCalc bool
	canConv bool
}{
	{"5 + ((1 + 2) * 4) - 3",
		[]string{"5", "1", "2", "+", "4", "*", "+", "3", "-"},
		big.NewRat(14, 1),
		true,
		true,
	},
	{"(1 + 2) * 3",
		[]string{"1", "2", "+", "3", "*"},
		big.NewRat(9, 1),
		true,
		true,
	},
	{"(1 + 2) / 0",
		[]string{"1", "2", "+", "0", "/"},
		nil,
		false, // zero division
		true,
	},
	{"(1 + 2 / 4",
		[]string{},
		nil,
		false,
		false,
	},
	{"1 / 2 + ( 2 + 3 ) * ( 9 - 2 * 2 - 3 / 4)",
		[]string{"1", "2", "/", "2", "3", "+", "9", "2", "2", "*", "-", "3", "4", "/", "-", "*", "+"},
		big.NewRat(87, 4),
		true,
		true,
	},
	{"-1.33",
		[]string{"1.33", "@"},
		big.NewRat(133, -100),
		true,
		true,
	},
	{"-1.5+2-2.5+3",
		[]string{"1.5", "@", "2", "+", "2.5", "-", "3", "+"},
		big.NewRat(1, 1),
		true,
		true,
	},
	{"sin(3**3)",
		[]string{"3", "3", "**", "sin"},
		big.NewRat(538391784348579, 562949953421312),
		true,
		true,
	},
	{"sin(2^3)",
		[]string{"2", "3", "^", "sin"},
		big.NewRat(4455673430828989, 4503599627370496),
		true,
		true,
	},
	{"tan(4÷-2×(8%6)+1.5)",
		[]string{"4", "2", "@", "÷", "8", "6", "%", "×", "1.5", "+", "tan"},
		big.NewRat(6728578678962965, 9007199254740992),
		true,
		true,
	},
	{"AbS(-1.5)",
		[]string{"1.5", "@", "AbS"},
		big.NewRat(3, 2),
		true,
		true,
	},
}

func TestRPN(t *testing.T) {
	for _, tc := range testCase {
		r, err := New(tc.in)
		if err != nil {
			if tc.canConv {
				t.Errorf("can not convert infix notation [%v], err %v", tc.in, err)
			}
			continue
		}
		if !equal(tc.postfix, r.Postfix()) {
			t.Errorf("infix [%v] postfix should be %v but %v", tc.in, tc.postfix, r.Postfix())
			continue
		}
		if result, err := r.Result(); err != nil {
			if !tc.canCalc {
				continue
			}
			t.Error(err)
			continue
		} else {
			if result.Cmp(tc.result) != 0 {
				t.Errorf("postfix %v result should be %v but %v", tc.postfix, tc.result, result)
				continue
			}
		}
	}
}

func BenchmarkRPN(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tc := range testCase {
			r, err := New(tc.in)
			if err != nil {
				continue
			}
			if !equal(tc.postfix, r.Postfix()) {
				continue
			}
			if result, err := r.Result(); err != nil {
				continue
			} else {
				if result.Cmp(tc.result) != 0 {
					continue
				}
			}
		}
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
