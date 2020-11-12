package rpn

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"text/scanner"
)

const (
	opOff = math.MaxInt8

	associativeLeft int8 = iota
	associativeRight
)

const (
	tokenTypeUnknown uint8 = 1 + iota
	tokenTypeOperand
	tokenTypeOperator
	tokenTypeParenthesis
	tokenTypeFunction
)

var (
	floatReg      = regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	funcReg       = regexp.MustCompile(`(?i)(abs|sin|cos|tan|ln|arcsin|arccos|arctan|sqrt)`)
	blankReg      = regexp.MustCompile(`\s+`)
	unaryMinusReg = regexp.MustCompile(`((?:^|[-+^%*/!~=(×÷])\s*)-`)
)

var (
	ErrUnrecognizedExpression = errors.New("unrecognized expression")
	ErrZeroDivision           = errors.New("zero division")
)

var (
	// operator precedence and operator associative
	operators = map[string][2]int8{
		"**": {opOff - 1, associativeLeft},
		"^":  {opOff - 1, associativeLeft},
		"@":  {opOff - 2, associativeRight}, // unary minus
		"*":  {opOff - 3, associativeLeft},
		"×":  {opOff - 3, associativeLeft},
		"/":  {opOff - 3, associativeLeft},
		"÷":  {opOff - 3, associativeLeft},
		"%":  {opOff - 3, associativeLeft},
		"+":  {opOff - 4, associativeLeft},
		"-":  {opOff - 4, associativeLeft},
	}
)

// RPN represents reverse Polish notation
type RPN struct {
	infix   []*token
	postfix []*token
	result  *big.Rat
}

// New new reverse Polish notation with a infix notation string pattern
func New(expr string) (*RPN, error) {
	infix := tokenise(expr)
	postfix, err := shuntingYard(infix)
	if err != nil {
		return nil, err
	}
	r := &RPN{
		infix:   infix,
		postfix: postfix,
	}
	return r, nil
}

// Result return the evaluate result from postfix notation
func (r *RPN) Result() (*big.Rat, error) {
	if r.result != nil {
		return r.result, nil
	}
	rv, err := calculate(r.postfix)
	if err != nil {
		return nil, err
	}
	r.result = rv
	return rv, nil
}

// Postfix postfix format output
func (r *RPN) Postfix() []string {
	s := make([]string, 0, len(r.postfix))
	for _, tok := range r.postfix {
		s = append(s, tok.v)
	}
	return s
}

type token struct {
	tp uint8
	v  string
}

func tokenise(expr string) []*token {
	expr = unaryMinusReg.ReplaceAllString(expr, "$1 @")
	expr = floatReg.ReplaceAllString(expr, " ${1} ")
	expr = funcReg.ReplaceAllString(expr, " ${1} ")
	expr = strings.Replace(expr, "(", " ( ", -1)
	expr = strings.Replace(expr, ")", " ) ", -1)
	expr = blankReg.ReplaceAllString(strings.TrimSpace(expr), "|")
	rs := strings.Split(expr, "|")

	tokens := make([]*token, 0, len(rs))
	for _, tok := range rs {
		tokens = append(tokens, &token{
			tp: typeOfToken(tok),
			v:  tok,
		})
	}
	return tokens
}

func typeOfToken(tok string) uint8 {
	if floatReg.MatchString(tok) {
		return tokenTypeOperand
	} else if funcReg.MatchString(tok) {
		return tokenTypeFunction
	} else if tok == "(" || tok == ")" {
		return tokenTypeParenthesis
	} else if _, ok := operators[tok]; ok {
		return tokenTypeOperator
	} else {
		return tokenTypeUnknown
	}
}

func shuntingYard(input []*token) ([]*token, error) {
	output := make([]*token, 0, len(input))
	ops := make([]*token, 0, len(input)) // stack for operator
	parens := [2]int{0, 0}
	for i := 0; i < len(input); i++ {
		t := input[i]
		switch t.tp {
		case tokenTypeUnknown:
			return nil, ErrUnrecognizedExpression
		case tokenTypeOperand:
			output = append(output, t)
		case tokenTypeFunction:
			ops = append(ops, t)
		case tokenTypeOperator:
			if _, ok := operators[t.v]; !ok {
				return nil, ErrUnrecognizedExpression
			}
			op1 := t
			for len(ops) > 0 {
				as1 := operators[op1.v][1]
				op2 := ops[len(ops)-1]
				if (priorityLE(op1.v, op2.v) && as1 == associativeLeft) || (!priorityGT(op1.v, op2.v) && as1 == associativeRight) {
					output = append(output, op2)
					ops = ops[:len(ops)-1]
					continue
				}
				break
			}
			ops = append(ops, op1)
		case tokenTypeParenthesis:
			switch t.v {
			case "(":
				ops = append(ops, t)
				parens[0]++
			case ")":
				parens[1]++
				mismatch := true
				for len(ops) > 0 {
					top := ops[len(ops)-1]
					if top.v != "(" {
						output = append(output, top)
						ops = ops[:len(ops)-1]
						continue
					}
					mismatch = false
					ops = ops[:len(ops)-1]
					break
				}
				if mismatch {
					return nil, ErrUnrecognizedExpression
				}
			}
		}
	}

	if parens[0] != parens[1] {
		return nil, ErrUnrecognizedExpression
	}

	if len(ops) > 0 {
		top := ops[len(ops)-1]
		if top.v == "(" || top.v == ")" {
			return nil, ErrUnrecognizedExpression
		}
	}

	for i := len(ops) - 1; i >= 0; i-- {
		output = append(output, ops[i])
	}

	return output, nil
}

func priorityLE(op1, op2 string) bool {
	return operators[op1][0] <= operators[op2][0]
}

func priorityGT(op1, op2 string) bool {
	return operators[op1][0] > operators[op2][0]
}

func calculate(postfix []*token) (*big.Rat, error) {
	var stack []*big.Rat
	for _, tok := range postfix {
		switch tok.tp {
		case tokenTypeUnknown, tokenTypeParenthesis:
			return nil, ErrUnrecognizedExpression
		case tokenTypeOperand:
			tmp := new(big.Rat)
			if _, err := fmt.Sscan(tok.v, tmp); err != nil {
				return nil, err
			}
			stack = append(stack, tmp)
		case tokenTypeOperator:
			tmp := new(big.Rat)
			if len(stack) == 0 {
				return nil, ErrUnrecognizedExpression
			}
			op2 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if tok.v == "@" {
				stack = append(stack, tmp.Mul(big.NewRat(-1, 1), op2))
				continue
			}
			if len(stack) == 0 {
				return nil, ErrUnrecognizedExpression
			}
			op1 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch tok.v {
			case "+":
				stack = append(stack, tmp.Add(op1, op2))
			case "-":
				stack = append(stack, tmp.Sub(op1, op2))
			case "*", "×":
				stack = append(stack, tmp.Mul(op1, op2))
			case "/", "÷":
				if f, _ := op2.Float64(); f == 0 {
					return nil, ErrZeroDivision
				}
				stack = append(stack, tmp.Quo(op1, op2))
			case "%":
				f1, _ := op1.Float64()
				f2, _ := op2.Float64()
				stack = append(stack, tmp.SetFloat64(math.Mod(f1, f2)))
			case "**", "^":
				f1, _ := op1.Float64()
				f2, _ := op2.Float64()
				stack = append(stack, tmp.SetFloat64(math.Pow(f1, f2)))

			default:
				return nil, ErrUnrecognizedExpression
			}
		case tokenTypeFunction:
			if len(stack) == 0 {
				return nil, ErrUnrecognizedExpression
			}
			tmp := new(big.Rat)
			op := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			fn := strings.ToLower(tok.v)
			f, _ := op.Float64()
			switch fn {
			case "abs":
				stack = append(stack, tmp.SetFloat64(math.Abs(f)))
			case "sin":
				stack = append(stack, tmp.SetFloat64(math.Sin(f)))
			case "cos":
				stack = append(stack, tmp.SetFloat64(math.Cos(f)))
			case "tan":
				stack = append(stack, tmp.SetFloat64(math.Tan(f)))
			case "ln":
				stack = append(stack, tmp.SetFloat64(math.Log(f)))
			case "arcsin":
				stack = append(stack, tmp.SetFloat64(math.Asin(f)))
			case "arccos":
				stack = append(stack, tmp.SetFloat64(math.Acos(f)))
			case "arctan":
				stack = append(stack, tmp.SetFloat64(math.Atan(f)))
			case "sqrt":
				stack = append(stack, tmp.SetFloat64(math.Sqrt(f)))
			default:
				return nil, ErrUnrecognizedExpression
			}
		}
	}

	if len(stack) == 0 {
		return nil, ErrUnrecognizedExpression
	}
	rv := stack[len(stack)-1]
	return rv, nil
}

func scan(expr string) []*token {
	var s scanner.Scanner
	s.Init(strings.NewReader(expr))
	tokens := make([]*token, 0)
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		var t token
		if tok == scanner.Int || tok == scanner.Float {
			t.tp = tokenTypeOperand
		} else {
			t.tp = tokenTypeOperator
		}
		t.v = s.TokenText()
		tokens = append(tokens, &t)
	}
	return tokens
}
