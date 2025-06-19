package main
import (
	"testing"
)

func TestCleanInput(t *testing.T) {

	cases := []struct {
		input string
		expected []string
	} {
		{
		input: "  hello  world  ",
		expected: []string{"hello","world"},
		},
		{
		input: "\thello\tworl\td",
		expected: []string{"hello","worl","d"},
		},
		{
		input: "\nhello\n\tworld\n,\t,",
		expected: []string{"hello","world",",",","},
		},
	}

	for _,c := range cases {
		actual:=cleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("Lengths don't match, EXPECTED: %v\tACTUAL: %v",c.expected,actual)
		}
		for i := range actual { //iterate over words in strings
			word := actual[i]
			expectedWord := c.expected[i]
			if (word != expectedWord) {
				t.Errorf("Words don't match, EXPECTED: %v\tACTUAL:%v",c.expected,actual)
			}
		}
	}
}