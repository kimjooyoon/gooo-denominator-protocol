package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-denominator-protocol/internal/denominator"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "digest":
		err = digestCommand(os.Args[2:])
	case "evaluate":
		err = evaluateCommand(os.Args[2:])
	case "migrate":
		err = migrateCommand(os.Args[2:])
	case "improvement":
		err = improvementCommand(os.Args[2:])
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func digestCommand(args []string) error {
	flags := flag.NewFlagSet("digest", flag.ContinueOnError)
	contractPath := flags.String("contract", "", "contract JSON path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var contract denominator.Denominator
	if err := denominator.DecodeFile(*contractPath, &contract); err != nil {
		return err
	}
	digest, err := denominator.ContractDigest(contract)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, digest)
	return err
}

func evaluateCommand(args []string) error {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	contractPath := flags.String("contract", "", "contract JSON path")
	evidencePath := flags.String("evidence", "", "evidence JSON path")
	outputPath := flags.String("output", "", "report output path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var contract denominator.Denominator
	if err := denominator.DecodeFile(*contractPath, &contract); err != nil {
		return emitFailure(*outputPath, "CONTRACT_JSON_INVALID", err)
	}
	var evidence denominator.Evidence
	if err := denominator.DecodeFile(*evidencePath, &evidence); err != nil {
		report := denominator.Evaluate(contract, evidence)
		if writeErr := emit(*outputPath, report); writeErr != nil {
			return writeErr
		}
		return err
	}
	report := denominator.Evaluate(contract, evidence)
	if err := emit(*outputPath, report); err != nil {
		return err
	}
	if report.Decision == denominator.Refuted {
		return fmt.Errorf("evaluation is REFUTED")
	}
	return nil
}

func migrateCommand(args []string) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fromPath := flags.String("from", "", "source contract JSON path")
	toPath := flags.String("to", "", "target contract JSON path")
	receiptPath := flags.String("receipt", "", "migration receipt JSON path")
	outputPath := flags.String("output", "", "report output path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var from, to denominator.Denominator
	var receipt denominator.MigrationReceipt
	if err := denominator.DecodeFile(*fromPath, &from); err != nil {
		return emitMigrationFailure(*outputPath, "FROM_CONTRACT_JSON_INVALID", err)
	}
	if err := denominator.DecodeFile(*toPath, &to); err != nil {
		return emitMigrationFailure(*outputPath, "TO_CONTRACT_JSON_INVALID", err)
	}
	if err := denominator.DecodeFile(*receiptPath, &receipt); err != nil {
		return emitMigrationFailure(*outputPath, "RECEIPT_JSON_INVALID", err)
	}
	report := denominator.ValidateMigration(from, to, receipt)
	if err := emit(*outputPath, report); err != nil {
		return err
	}
	if report.Decision == denominator.Refuted {
		return fmt.Errorf("migration is REFUTED")
	}
	return nil
}

func improvementCommand(args []string) error {
	flags := flag.NewFlagSet("improvement", flag.ContinueOnError)
	pairPath := flags.String("pair", "", "improvement pair JSON path")
	outputPath := flags.String("output", "", "report output path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var pair denominator.ImprovementPair
	if err := denominator.DecodeFile(*pairPath, &pair); err != nil {
		return emitImprovementFailure(*outputPath, "PAIR_JSON_INVALID", err)
	}
	report := denominator.AssessImprovement(pair)
	if err := emit(*outputPath, report); err != nil {
		return err
	}
	return nil
}

func emitFailure(path, reason string, err error) error {
	failure := map[string]any{
		"schema":      denominator.ReportSchema,
		"decision":    denominator.Refuted,
		"fail_closed": true,
		"claim": map[string]any{
			"state": denominator.Refuted, "stage": "VALIDATION", "step": "DECODE_JSON",
			"reason": reason, "next_operation": "RESTORE_VALID_JSON", "blocked_by": []string{},
		},
		"error": err.Error(),
	}
	if writeErr := emit(path, failure); writeErr != nil {
		return writeErr
	}
	return err
}

func emitMigrationFailure(path, reason string, err error) error {
	failure := denominator.MigrationReport{
		Schema: denominator.ReceiptSchema, Decision: denominator.Refuted, FailClosed: true,
		Claim: denominator.Claim{State: denominator.Refuted, Stage: "VALIDATION", Step: "DECODE_JSON", Reason: reason, NextOperation: "RESTORE_VALID_JSON", BlockedBy: []string{}},
	}
	if writeErr := emit(path, failure); writeErr != nil {
		return writeErr
	}
	return err
}

func emitImprovementFailure(path, reason string, err error) error {
	failure := denominator.ImprovementReport{
		Schema: denominator.PairSchema, Decision: denominator.Unknown,
		Claim: denominator.Claim{State: denominator.Unknown, Stage: "VALIDATION", Step: "DECODE_JSON", Reason: reason, UnknownClass: "DIRECT_MISSING", NextOperation: "RESTORE_VALID_JSON", BlockedBy: []string{}},
	}
	if writeErr := emit(path, failure); writeErr != nil {
		return writeErr
	}
	return err
}

func emit(path string, value any) error {
	if path == "" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	return denominator.WriteJSON(path, value)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: denominator digest|evaluate|migrate|improvement")
}
