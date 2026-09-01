package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-self-rewrite-sandbox/internal/sandbox"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: gooo-self-rewrite-sandbox run --meta PATH --input-root PATH --phase PATH --source PATH --corpus PATH --work PATH --report PATH")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	options := sandbox.Options{}
	flags.StringVar(&options.MetaPath, "meta", "", "authoritative .gooo actuation contract")
	flags.StringVar(&options.InputRoot, "input-root", "", "immutable compiler input root")
	flags.StringVar(&options.PhasePath, "phase", "", "baseline compiler phase inside input-root")
	flags.StringVar(&options.SourcePath, "source", "", "source .gooo inside input-root")
	flags.StringVar(&options.CorpusPath, "corpus", "", "fixed fixture corpus")
	flags.StringVar(&options.WorkDir, "work", "", "caller-owned temporary output workspace")
	flags.StringVar(&options.ReportPath, "report", "", "report output path")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	report, err := sandbox.Run(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(data))
	if report.Decision != "CLOSED" {
		os.Exit(1)
	}
}
