package commands

import (
	"strings"
)

type stringsFlag struct {
	target *[]string
}

func (s *stringsFlag) String() string {
	return strings.Join(*s.target, ",")
}

func (s *stringsFlag) Set(value string) error {
	*s.target = append(*s.target, value)
	return nil
}
