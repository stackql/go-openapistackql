package fuzzymatch

type FuzzyMatcher[T any] interface {
	Find(T) (T, bool)
}
