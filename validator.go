package main

import "regexp"

type Constraint interface {
	GetValue() interface{}
	GetErrorMessage() string
	Validate() bool
}

type ExactLength struct {
	Value        string
	ErrorMessage string
	Length       int
}

func (exlen ExactLength) Validate() bool {
	return len(exlen.Value) == exlen.Length
}

func (exlen ExactLength) GetValue() interface{} {
	return exlen.Value
}

func (exlen ExactLength) GetErrorMessage() string {
	return exlen.ErrorMessage
}

type Match struct {
	Value        string
	ErrorMessage string
	Regex        *regexp.Regexp
}

func (m Match) Validate() bool {
	return m.Regex.Match([]byte(m.Value))
}

func (m Match) GetValue() interface{} {
	return m.Value
}

func (m Match) GetErrorMessage() string {
	return m.ErrorMessage
}
