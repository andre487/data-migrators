package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/akamensky/argparse"

	"github.com/andre487/data-migrators/providers/fatsecret"
	"github.com/andre487/data-migrators/utils/secrets"
)

func main() {
	args := getArgs()
	switch args.Action {
	case "get-fatsecret-diary":
		actionGetFatsecretDiary(args)
		break
	default:
		log.Fatalf("Unknown action %s\n", args.Action)
	}
}

func actionGetFatsecretDiary(args cliArgs) {
	cmdArgs := args.ActionArgs.(fsDiaryArgs)
	keyData, err := secrets.GetSecretFromFile(cmdArgs.KeyFilePath)
	if err != nil {
		log.Fatal(err)
	}
	fs, err := fatsecret.New(keyData)
	if err != nil {
		log.Fatal(err)
	}
	res, err := fs.GetDiary(cmdArgs.FromDate, cmdArgs.ToDate)
	if err != nil {
		log.Fatal(err)
	}

	jsRes, err := json.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}
	_, err = cmdArgs.OutFile.Write(jsRes)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("FatSecret: diary was written to file %s", cmdArgs.OutFile.Name())
}

type cliArgs struct {
	Action     string
	ActionArgs interface{}
}

type fsDiaryArgs struct {
	KeyFilePath string
	OutFile     *os.File
	FromDate    time.Time
	ToDate      time.Time
}

func getArgs() cliArgs {
	parser := argparse.NewParser("data-migrations", "Migrate data for andre487")

	getFsDiaryCommand := parser.NewCommand("get-fatsecret-diary", "Get FatSecret diary")
	fsDiaryOutFile := getFsDiaryCommand.FilePositional(os.O_CREATE|os.O_WRONLY, 0644, &argparse.Options{
		Default: "fat-secret-diary-data.json",
	})
	fsKeyFilePath := getFsDiaryCommand.String("k", "key-file", &argparse.Options{
		Default: "~/.tokens/fatsecret.json",
	})
	fsDiaryFromDate := getFsDiaryCommand.String("m", "from-date", &argparse.Options{
		Default:  time.Now().AddDate(0, 0, -2).Format("2006-01-01"),
		Validate: validateDate,
	})
	fsDiaryToDate := getFsDiaryCommand.String("t", "to-date", &argparse.Options{
		Default:  time.Now().Format("2006-01-01"),
		Validate: validateDate,
	})

	helpCommand := parser.NewCommand("help", "Show help")

	rootUsage := parser.Usage("")
	if err := parser.Parse(os.Args); err != nil {
		fmt.Println(parser.Usage(err))
		os.Exit(1)
	}

	if helpCommand.Happened() {
		fmt.Println(rootUsage)
		os.Exit(0)
	}

	res := cliArgs{}
	switch {
	case getFsDiaryCommand.Happened():
		res.Action = getFsDiaryCommand.GetName()
		res.ActionArgs = fsDiaryArgs{
			KeyFilePath: *fsKeyFilePath,
			OutFile:     fsDiaryOutFile,
			FromDate:    parseDate(fsDiaryFromDate),
			ToDate:      parseDate(fsDiaryToDate),
		}
		break
	}

	return res
}

var dateRe, _ = regexp.Compile("^\\d{4}-\\d{2}-\\d{2}$")

func validateDate(val []string) error {
	if !dateRe.Match([]byte(val[0])) {
		return errors.New(fmt.Sprintf("date should be YYYY-MM-DD, not %s", val[0]))
	}
	return nil
}

func parseDate(dt *string) time.Time {
	res, err := time.Parse("2006-01-01", *dt)
	if err != nil {
		log.Fatal(err)
	}
	return res
}
