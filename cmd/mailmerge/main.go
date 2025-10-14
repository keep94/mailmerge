package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/keep94/mailmerge/merge"
	"github.com/keep94/toolbox/build"
	"github.com/keep94/toolbox/mailer"
	"gopkg.in/yaml.v3"
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
	csvFile, err := merge.ReadCsv(fCsv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	csvFile = csvFile.SelectGoing()
	template, err := readTemplate(fTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if fEmails != "" {
		var err error
		csvFile, err = doEmailFilter(csvFile, fEmails)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else if fNoEmails != "" {
		var err error
		csvFile, err = doNoEmailFilter(csvFile, fNoEmails)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	sender := createEmailSender(config, fDryRun)
	defer sender.Shutdown()
	for index, row := range csvFile.Rows {
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
	row merge.CsvRow,
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

func doEmailFilter(csvFile *merge.CsvFile, emails string) (
	*merge.CsvFile, error) {
	selectedEmails := merge.NewEmailSet(emails)
	if err := checkEmails(csvFile, selectedEmails); err != nil {
		return nil, err
	}
	return csvFile.SelectEmails(selectedEmails), nil
}

func doNoEmailFilter(csvFile *merge.CsvFile, noEmails string) (
	*merge.CsvFile, error) {
	selectedNoEmails := merge.NewEmailSet(noEmails)
	if err := checkEmails(csvFile, selectedNoEmails); err != nil {
		return nil, err
	}
	return csvFile.SelectNoEmails(selectedNoEmails), nil
}

func checkEmails(csvFile *merge.CsvFile, emails merge.EmailSet) error {
	unrecognizedEmails := emails.Difference(csvFile.AsEmailSet())
	if len(unrecognizedEmails) > 0 {
		return fmt.Errorf("Unrecognized emails: %s", unrecognizedEmails)
	}
	return nil
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
