package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type ArrayPredicateTestSuite struct {
	parsertest.Suite
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateErrors() {
	s.RETC("", `expected an opening '\['`, true)
	s.RETC("]", `expected an opening '\['`, false)
	s.RETC("f", `expected an opening '\['`, true)
	s.RETC("[", `expected a closing '\]'`, false)
	s.RETC("[a", `expected a closing '\]'`, false)
	s.RETC("[]", `expected a '\*', '\?', or an array index inside '\[\]'`, false)
	s.RETC("[*a]", `expected a closing '\]' after '\*'`, false)
	s.RETC("[?a]", `expected a closing '\]' after '\?'`, false)
	s.RETC("[a]", `expected an array index inside '\[\]'`, false)
	s.RETC("[-15]", `expected an array index inside '\[\]'`, false)
	s.RETC("[?]-true", `expected a '\.' or '\[' after '\]' but got -true instead`, false)
	s.RETC("[?]", `expected a predicate after \[\?\]`, false)
	s.RETC("[*]", `expected a predicate after \[\*\]`, false)
	s.RETC("[15]", `expected a predicate after \[15\]`, false)
	s.RETC("[?] +{", "expected.*closing.*}", false)
	// Test predicate expression errors
	s.RETC("[?] )", `\): no beginning '\('`, false)
	s.RETC("[?] (", `\(: missing closing '\)'`, false)
	s.RETC("[?] ( -true", `\(: missing closing '\)'`, false)
	s.RETC("[?] ( )", `\(\): empty inner expression`, false)
	s.RETC("[?] ( -true -false -foo", "unknown predicate -foo", false)
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	// Test -empty
	s.RTC("-empty", "", []interface{}{})
	// Test each of the possible arrayPs
	s.RTC("[?] -true -size", "-size", toA(false, true))
	s.RTC("[*] -true -size", "-size", toA(true, true))
	s.RTC("[0] -true -size", "-size", toA(true))
	// Test key sequences
	s.RTC("[?][?] -true -size", "-size", toA(toA(true)))
	s.RTC("[?].key -true -size", "-size", toA(mp))
	// Now test predicate expressions. The predicate expression parser's
	// already well tested, so these are just some sanity checks.
	s.RNTC("[0] ( -true -a -false ) -size", "-size", toA(true))
	s.RTC("[0] ( -true -o -false ) -size", "-size", toA(true))
	s.RTC("[0] ( ! -false ) -size", "-size", toA(true))
	s.RTC("[0] ( ! ( -true -a -false ) ) -size", "-size", toA(true))
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateType() {
	// These test only the valid inputs. The error cases are tested in
	// TestParseArrayPredicateErrors.

	ptype, token, err := parseArrayPredicateType("[?]")
	if s.NoError(err) {
		s.Equal(byte('s'), ptype.t)
		s.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[*]")
	if s.NoError(err) {
		s.Equal(byte('a'), ptype.t)
		s.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[15]")
	if s.NoError(err) {
		s.Equal(byte('n'), ptype.t)
		s.Equal(uint(15), ptype.n)
		s.Equal("", token)
	}
}

func (s *ArrayPredicateTestSuite) TestArrayPSome() {
	p := arrayP(arrayPredicateType{t: 's'}, trueP)

	// Returns false for a non-array value. Not(p) should also
	// return false.
	s.False(p.IsSatisfiedBy("foo"))
	s.False(p.Negate().IsSatisfiedBy("foo"))

	// Returns false if no elements satisfy p. Not(p) should return
	// true here.
	s.False(p.IsSatisfiedBy(toA(false, false)))
	s.True(p.Negate().IsSatisfiedBy(toA(false, false)))

	// Returns true if some element satifies . Not(p) should return
	// false here.
	s.True(p.IsSatisfiedBy(toA(true, false)))
	s.False(p.Negate().IsSatisfiedBy(toA(true, false)))
}

func (s *ArrayPredicateTestSuite) TestArrayPAll() {
	p := arrayP(arrayPredicateType{t: 'a'}, trueP)

	// Returns false for a non-array value. Not(p) should also
	// return false.
	s.False(p.IsSatisfiedBy("foo"))
	s.False(p.Negate().IsSatisfiedBy("foo"))

	// Returns false if no elements satisfy p. Not(p) should return
	// true here.
	s.False(p.IsSatisfiedBy(toA(false)))
	s.True(p.Negate().IsSatisfiedBy(toA(false)))

	// Returns false if only some of the elements satisfy p. Not(p) should
	// return true here
	s.False(p.IsSatisfiedBy(toA(false, true)))
	s.True(p.Negate().IsSatisfiedBy(toA(false, true)))

	// Returns true if all the elements satisfy P. Not(p) should return
	// false here.
	s.True(p.IsSatisfiedBy(toA(true)))
	s.False(p.Negate().IsSatisfiedBy(toA(true)))
	s.True(p.IsSatisfiedBy(toA(true, true)))
	s.False(p.Negate().IsSatisfiedBy(toA(true, true)))
}

func (s *ArrayPredicateTestSuite) TestArrayPNth() {
	p := arrayP(arrayPredicateType{t: 'n', n: 1}, trueP)

	// Returns false for a non-array value. Not(p) should also
	// return false.
	s.False(p.IsSatisfiedBy("foo"))
	s.False(p.Negate().IsSatisfiedBy("foo"))

	// Returns false if n >= len(vs). Not(p) should also return
	// false.
	s.False(p.IsSatisfiedBy(toA(true)))
	s.False(p.Negate().IsSatisfiedBy(toA(true)))

	// Returns false if vs[n] does not satisfy p. Not(p) should return
	// true here.
	s.False(p.IsSatisfiedBy(toA(true, false)))
	s.True(p.Negate().IsSatisfiedBy(toA(true, false)))

	// Returns true if vs[n] satisfies p. Not(p) should return false.
	s.True(p.IsSatisfiedBy(toA(true, true)))
	s.False(p.Negate().IsSatisfiedBy(toA(true, true)))
}

func toA(vs ...interface{}) []interface{} {
	return vs
}

func TestArrayPredicate(t *testing.T) {
	s := new(ArrayPredicateTestSuite)
	s.Parser = predicate.ToParser(parseArrayPredicate)
	suite.Run(t, s)
}
