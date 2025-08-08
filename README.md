# mailmerge

Does mailmerge on gmail.

This program merges a text file that uses the Go template language with a
CSV file. At a minimum, the CSV file must contain an "email" column and a
"name" column.

This program expects a .mailmerge.yaml file to be in your $HOME directory.
The format of .mailmerge.yaml looks like this:

```
emailId: gamilId
password: app_password
```

Run the program like this:

```
mailmerge -template template.txt -csv data.csv -subject "Your Email Subject"
```

template.txt may look like this:

```
Dear {{.name}}:

Your pet, {{.petname}} is due for a checkup.
```

data.csv may look like this:

```
name,email,petname
Alice,alice@gmail.com,Patches
Bob,bob@gmail.com,Rufus
```

As the job runs, it prints to stdout the index, email address, and name for the email currently being sent.

## Optional flags
- The -dryrun flag sends no emails, but prints to stdout the emails that would be sent.
- The -emails flag, if present, mail merges to the comma separated emails rather than the entire batch.
- The -noemails flag, if present, mail merges to all emails except the comma separated emails. If the -emails flag is present, -noemails is ignored.
- In case the program terminated early from an error, the -index flag can start the mailmerge job where it left off rather than at the beginning. e.g -index 3 starts the job at the email with index 3.
