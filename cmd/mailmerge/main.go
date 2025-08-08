package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/keep94/toolbox/build"
	"github.com/keep94/toolbox/mailer"
	"gopkg.in/yaml.v3"
)

const (
	kName  = "name"
	kEmail = "email"
)

var (
	fTemplate string
	fCsv      string
	fSubject  string
	fDryRun   bool
	fIndex    int
	fEmails   string
	fNoEmails string
	fVersion  bool
)

func main() {
	flag.Parse()
	if fVersion {
		version, _ := build.MainVersion()
		fmt.Println(build.BuildId(version))
		return
	}
	if fTemplate == "" || fCsv == "" || fSubject == "" {
		fmt.Println("-template, -csv, and -subject flags required.")
		flag.Usage()
		os.Exit(2)
	}
	config, err := readConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	csvRows, err := readCsv(fCsv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	template, err := readTemplate(fTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if fEmails != "" {
		var err error
		csvRows, err = doEmailFilter(csvRows, fEmails)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else if fNoEmails != "" {
		var err error
		csvRows, err = doNoEmailFilter(csvRows, fNoEmails)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	sender := createEmailSender(config, fDryRun)
	defer sender.Shutdown()
	for index, row := range csvRows {
		if index < fIndex {
			continue
		}
		fmt.Printf("%d %s %s\n", index, row.Email(), row.Name())
		email, err := createEmail(template, row, fSubject)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = <-sender.SendFuture(*email)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func createEmailSender(config *config, dryRun bool) emailSender {
	if dryRun {
		return dryRunMailer{}
	}
	return mailer.NewWithOptions(
		config.EmailId,
		config.Password,
		mailer.SendWaitTime(100*time.Millisecond),
	)
}

type dryRunMailer struct {
}

func (d dryRunMailer) SendFuture(email mailer.Email) <-chan error {
	fmt.Println()
	fmt.Println("To:", email.To)
	fmt.Println("Subject:", email.Subject)
	fmt.Println("Body:")
	fmt.Println(email.Body)
	result := make(chan error, 1)
	result <- nil
	close(result)
	return result
}

func (d dryRunMailer) Shutdown() {
}

func createEmail(
	template *template.Template,
	row csvRow,
	subject string) (*mailer.Email, error) {
	var builder strings.Builder
	if err := template.Execute(&builder, row); err != nil {
		return nil, err
	}
	result := &mailer.Email{
		Subject: subject,
		To:      []string{row.Email()},
		Body:    builder.String(),
	}
	return result, nil
}

type emailSender interface {
	SendFuture(email mailer.Email) <-chan error
	Shutdown()
}

func readTemplate(templatePath string) (*template.Template, error) {
	return template.ParseFiles(templatePath)
}

type csvRow map[string]string

func (c csvRow) Name() string {
	return c[kName]
}

func (c csvRow) Email() string {
	return c[kEmail]
}

type emailSet map[string]struct{}

func newEmailSet(commaSeparatedEmails string) emailSet {
	emailList := strings.Split(commaSeparatedEmails, ",")
	result := make(emailSet)
	for _, email := range emailList {
		result.Add(strings.TrimSpace(email))
	}
	return result
}

func (e emailSet) Contains(email string) bool {
	_, ok := e[email]
	return ok
}

func (e emailSet) Add(email string) {
	e[email] = struct{}{}
}

func (e emailSet) Difference(other emailSet) emailSet {
	result := make(emailSet)
	for email := range e {
		if !other.Contains(email) {
			result.Add(email)
		}
	}
	return result
}

func (e emailSet) String() string {
	emailSlice := make([]string, 0, len(e))
	for email := range e {
		emailSlice = append(emailSlice, email)
	}
	sort.Strings(emailSlice)
	return strings.Join(emailSlice, ", ")
}

func doEmailFilter(csvRows []csvRow, emails string) ([]csvRow, error) {
	selectedEmails := newEmailSet(emails)
	result, _, unrecognizedEmails := filterByEmails(csvRows, selectedEmails)
	if len(unrecognizedEmails) > 0 {
		return nil, fmt.Errorf("Unrecognized emails: %s", unrecognizedEmails)
	}
	return result, nil
}

func doNoEmailFilter(csvRows []csvRow, noEmails string) ([]csvRow, error) {
	selectedEmails := newEmailSet(noEmails)
	_, result, unrecognizedEmails := filterByEmails(csvRows, selectedEmails)
	if len(unrecognizedEmails) > 0 {
		return nil, fmt.Errorf("Unrecognized emails: %s", unrecognizedEmails)
	}
	return result, nil
}

func filterByEmails(csvRows []csvRow, emails emailSet) (
	selected, notSelected []csvRow, unrecognizedEmails emailSet) {
	foundEmails := make(emailSet)
	for _, row := range csvRows {
		if emails.Contains(row.Email()) {
			selected = append(selected, row)
			foundEmails.Add(row.Email())
		} else {
			notSelected = append(notSelected, row)
		}
	}
	unrecognizedEmails = emails.Difference(foundEmails)
	return
}

func readCsv(csvPath string) ([]csvRow, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readCsvFile(f)
}

func readCsvFile(r io.Reader) ([]csvRow, error) {
	csvReader := csv.NewReader(r)
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	var result []csvRow
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
	return result, nil
}

func createCsvRow(headers, row []string) csvRow {
	result := make(csvRow, len(headers))
	for index, colName := range headers {
		result[colName] = row[index]
	}
	return result
}

type config struct {
	EmailId  string `yaml:"emailId"`
	Password string `yaml:"password"`
}

func readConfig() (*config, error) {
	configPath := path.Join(os.Getenv("HOME"), ".mailmerge.yaml")
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var content bytes.Buffer
	if _, err := content.ReadFrom(f); err != nil {
		return nil, err
	}
	var result config
	if err := yaml.Unmarshal(content.Bytes(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func init() {
	flag.StringVar(&fTemplate, "template", "", "Path to template file")
	flag.StringVar(&fCsv, "csv", "", "Path to CSV file")
	flag.StringVar(&fSubject, "subject", "", "Subject")
	flag.BoolVar(&fDryRun, "dryrun", false, "Dry Run?")
	flag.IntVar(&fIndex, "index", 0, "Starting index")
	flag.StringVar(&fEmails, "emails", "", "Comma separated emails to include")
	flag.StringVar(
		&fNoEmails,
		"noemails",
		"",
		"Comma separated emails to exclude. Ignored if emails flag is present")
	flag.BoolVar(&fVersion, "version", false, "Show version")
}
