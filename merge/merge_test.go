package merge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	csvStr = `email,name,going
alice@gmail.com,alice,no
bob@gmail.com,bob,yes
charlie@gmail.com,charlie,yes
`
	csvStrNoGoingColumn = `email,name
alice@gmail.com,alice
bob@gmail.com,bob
charlie@gmail.com,charlie
`
	csvStrNoOneGoing = `email,name,going
alice@gmail.com,alice,n
bob@gmail.com,bob,n
charlie@gmail.com,charlie,n
`
)

func TestWithNotGoing(t *testing.T) {
	r := strings.NewReader(csvStr)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.WithNotGoing().write(&builder))
	assert.Equal(t, csvStrNoOneGoing, builder.String())
}

func TestWithNotGoingNoColumn(t *testing.T) {
	r := strings.NewReader(csvStrNoGoingColumn)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.WithNotGoing().write(&builder))
	assert.Equal(t, csvStrNoOneGoing, builder.String())
}

func TestSelectEmails(t *testing.T) {
	emails := NewEmailSet("alice@gmail.com,bob@gmail.com")
	r := strings.NewReader(csvStr)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.SelectEmails(emails).write(&builder))
	expected := `email,name,going
alice@gmail.com,alice,no
bob@gmail.com,bob,yes
`
	assert.Equal(t, expected, builder.String())
}

func TestSelectNoEmails(t *testing.T) {
	emails := NewEmailSet("bob@gmail.com")
	r := strings.NewReader(csvStr)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.SelectNoEmails(emails).write(&builder))
	expected := `email,name,going
alice@gmail.com,alice,no
charlie@gmail.com,charlie,yes
`
	assert.Equal(t, expected, builder.String())
}

func TestSelectGoing(t *testing.T) {
	r := strings.NewReader(csvStr)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.SelectGoing().write(&builder))
	expected := `email,name,going
bob@gmail.com,bob,yes
charlie@gmail.com,charlie,yes
`
	assert.Equal(t, expected, builder.String())
}

func TestSelectGoingNoGoingColumn(t *testing.T) {
	r := strings.NewReader(csvStrNoGoingColumn)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	var builder strings.Builder
	assert.NoError(t, csv.SelectGoing().write(&builder))
	expected := `email,name
alice@gmail.com,alice
bob@gmail.com,bob
charlie@gmail.com,charlie
`
	assert.Equal(t, expected, builder.String())
}

func TestIllegalRead(t *testing.T) {
	r := strings.NewReader("")
	_, err := readCsv(r)
	assert.Error(t, err)
}

func TestIllegalRead2(t *testing.T) {
	r := strings.NewReader(`email
alice@gmail.com
`)
	_, err := readCsv(r)
	assert.Error(t, err)
}

func TestAsEmailSet(t *testing.T) {
	r := strings.NewReader(csvStr)
	csv, err := readCsv(r)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"alice@gmail.com, bob@gmail.com, charlie@gmail.com",
		csv.AsEmailSet().String())
}

func TestDifference(t *testing.T) {
	lhs := NewEmailSet("alice@gmail.com,bob@gmail.com,charlie@gmail.com")
	rhs := NewEmailSet("alice@gmail.com,bob@gmail.com,echo@gmail.com")
	diff := lhs.Difference(rhs)
	assert.Equal(t, "charlie@gmail.com", diff.String())
}
