package merge

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
)

const (

	// The name column
	Name = "name"

	// The email column
	Email = "email"

	// The going column.
	Going = "going"
)

// CsvRow represents a single row of a mail merge CSV file. The keys
// are the column names; the values are the column values.
// CsvRow instances are designed to be immutable.
type CsvRow map[string]string

// Name returns the person's name
func (c CsvRow) Name() string {
	return c[Name]
}

// Email returns the person's email
func (c CsvRow) Email() string {
	return c[Email]
}

// Going returns if person is going to the event. True if it does not start
// with "n" or "N"
func (c CsvRow) Going() bool {
	return !strings.HasPrefix(strings.ToLower(c[Going]), "n")
}

// WithNotGoing returns a CsvRow like this one but with the going column
// set to "n"
func (c CsvRow) WithNotGoing() CsvRow {
	result := c.copy()
	result[Going] = "n"
	return result
}

func (c CsvRow) copy() CsvRow {
	result := make(CsvRow, len(c))
	for k, v := range c {
		result[k] = v
	}
	return result
}

// EmailSet represents a set of emails
type EmailSet map[string]struct{}

// NewEmailSet returns a new EmailSet from comma separated emails.
func NewEmailSet(commaSeparatedEmails string) EmailSet {
	emailList := strings.Split(commaSeparatedEmails, ",")
	result := make(EmailSet)
	for _, email := range emailList {
		result.Add(strings.TrimSpace(email))
	}
	return result
}

// Contains returns true if this instance contains email.
func (e EmailSet) Contains(email string) bool {
	_, ok := e[email]
	return ok
}

// Add adds an email to this instance in place.
func (e EmailSet) Add(email string) {
	e[email] = struct{}{}
}

// Difference returns the set of emails in e that are not in other.
func (e EmailSet) Difference(other EmailSet) EmailSet {
	result := make(EmailSet)
	for email := range e {
		if !other.Contains(email) {
			result.Add(email)
		}
	}
	return result
}

// String returns this instance as a comma separated list of emails
// sorted alphabetically.
func (e EmailSet) String() string {
	emailSlice := make([]string, 0, len(e))
	for email := range e {
		emailSlice = append(emailSlice, email)
	}
	sort.Strings(emailSlice)
	return strings.Join(emailSlice, ", ")
}

// CsvFile represents a mail merge CsvFile. CsvFile instances are designed
// to be immutable.
type CsvFile struct {

	// The headers
	Headers []string

	// The rows
	Rows []CsvRow
}

// SelectEmails returns a CsvFile like this instance that contains
// only the rows with emails that are in emails.
func (c *CsvFile) SelectEmails(emails EmailSet) *CsvFile {
	f := func(row CsvRow) bool {
		return emails.Contains(row.Email())
	}
	return c.sel(f)
}

// SelectNoEmails returns a CsvFile like this instance that contains
// only the rows with emails that are not in emails.
func (c *CsvFile) SelectNoEmails(emails EmailSet) *CsvFile {
	f := func(row CsvRow) bool {
		return !emails.Contains(row.Email())
	}
	return c.sel(f)
}

// SelectGoing returns a CsvFile like this instance that contains
// only the rows that are going to the event.
func (c *CsvFile) SelectGoing() *CsvFile {
	f := func(row CsvRow) bool {
		return row.Going()
	}
	return c.sel(f)
}

// AsEmailSet returns this instance as an EmailSet.
func (c *CsvFile) AsEmailSet() EmailSet {
	result := make(EmailSet, len(c.Rows))
	for _, row := range c.Rows {
		result.Add(row.Email())
	}
	return result
}

// WithNotGoing returns a CsvFile like this instance where every row has
// going set to "n"
func (c *CsvFile) WithNotGoing() *CsvFile {
	result := make([]CsvRow, 0, len(c.Rows))
	for _, row := range c.Rows {
		result = append(result, row.WithNotGoing())
	}
	return &CsvFile{Headers: addGoingHeader(c.Headers), Rows: result}
}

// Write writes this instance to a file.
func (c *CsvFile) Write(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.write(f)
}

func (c *CsvFile) sel(f func(CsvRow) bool) *CsvFile {
	var result []CsvRow
	for _, row := range c.Rows {
		if f(row) {
			result = append(result, row)
		}
	}
	return &CsvFile{Headers: c.Headers, Rows: result}
}

func (c *CsvFile) write(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	if err := csvWriter.Write(c.Headers); err != nil {
		return err
	}
	csvRow := make([]string, 0, len(c.Headers))
	for _, row := range c.Rows {
		for _, header := range c.Headers {
			csvRow = append(csvRow, row[header])
		}
		if err := csvWriter.Write(csvRow); err != nil {
			return err
		}
		csvRow = csvRow[:0]
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

// ReadCsv reads a CsvFile.
func ReadCsv(csvPath string) (*CsvFile, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readCsv(f)
}

func readCsv(r io.Reader) (*CsvFile, error) {
	csvReader := csv.NewReader(r)
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	var result []CsvRow
	row, err := csvReader.Read()
	for err != io.EOF {
		if err != nil {
			return nil, err
		}
		lineNo, _ := csvReader.FieldPos(0)
		crow := createCsvRow(headers, row)
		if crow.Name() == "" || crow.Email() == "" {
			err = fmt.Errorf(
				"Line %d: name and email columns must be present", lineNo)
			return nil, err
		}
		result = append(result, crow)
		row, err = csvReader.Read()
	}
	return &CsvFile{Headers: headers, Rows: result}, nil
}

func createCsvRow(headers, row []string) CsvRow {
	result := make(CsvRow, len(headers))
	for index, colName := range headers {
		result[colName] = row[index]
	}
	return result
}

func addGoingHeader(headers []string) []string {
	if slices.Contains(headers, Going) {
		return headers
	}
	result := make([]string, 0, len(headers)+1)
	result = append(result, headers...)
	return append(result, Going)
}
