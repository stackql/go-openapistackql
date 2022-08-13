package fuzzymatch

import (
	"regexp"
)

type StringFuzzyPair struct {
	r *regexp.Regexp
	s string
}

func NewFuzzyPair(r *regexp.Regexp, s string) StringFuzzyPair {
	return StringFuzzyPair{
		r: r,
		s: s,
	}
}

type StringFuzzyMatcher struct {
	matchers []StringFuzzyPair
}

func (fm *StringFuzzyMatcher) Find(s string) (string, bool) {
	for _, m := range fm.matchers {
		if m.r.MatchString(s) {
			return m.s, true
		}
	}
	return "", false
}

func NewRegexpStringMetcher(matchers []StringFuzzyPair) FuzzyMatcher[string] {
	return &StringFuzzyMatcher{
		matchers: matchers,
	}
}
