package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/yzchan/iploc"
)

var version = "dev"

type queryResult struct {
	IP      string `json:"ip"`
	StartIP string `json:"start_ip,omitempty"`
	StopIP  string `json:"stop_ip,omitempty"`
	RecordA string `json:"record_a,omitempty"`
	RecordB string `json:"record_b,omitempty"`
	Error   string `json:"error,omitempty"`
}

type cliConfig struct {
	dbPath      string
	format      string
	useMap      bool
	failOnError bool
	showVersion bool
	ips         []string
}

func main() {
	cfg := parseFlags(os.Args[1:])
	if err := run(os.Stdout, os.Stderr, cfg); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, usageErr.Error())
			printUsage(os.Stderr)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFlags(args []string) cliConfig {
	fs := flag.NewFlagSet("iploc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := cliConfig{}
	fs.StringVar(&cfg.dbPath, "db", "", "path to qqwry.dat database")
	fs.StringVar(&cfg.format, "format", "text", "output format: text, json, jsonl")
	fs.BoolVar(&cfg.useMap, "map", false, "preload records into a map before querying")
	fs.BoolVar(&cfg.failOnError, "fail-on-error", false, "exit with code 1 if any input IP fails")
	fs.BoolVar(&cfg.showVersion, "version", false, "print version and exit")
	_ = fs.Parse(args)
	cfg.ips = fs.Args()
	return cfg
}

func run(stdout io.Writer, stderr io.Writer, cfg cliConfig) error {
	if cfg.showVersion {
		fmt.Fprintf(stdout, "iploc %s\n", version)
		return nil
	}
	if cfg.dbPath == "" {
		return usageError("missing required -db")
	}
	if cfg.format != "text" && cfg.format != "json" && cfg.format != "jsonl" {
		return usageError("invalid -format: " + cfg.format)
	}

	ips := cfg.ips
	if len(ips) == 0 {
		stdinIPs, err := readIPs(os.Stdin)
		if err != nil {
			return err
		}
		ips = stdinIPs
	}
	if len(ips) == 0 {
		return usageError("missing IPv4 address")
	}

	parser, err := iploc.NewQQWryParser(cfg.dbPath)
	if err != nil {
		return fmt.Errorf("open database failed: %w", err)
	}
	if cfg.useMap {
		if err := parser.FormatMap(); err != nil {
			return fmt.Errorf("format map failed: %w", err)
		}
	}

	results := make([]queryResult, 0, len(ips))
	hasError := false
	for _, rawIP := range ips {
		result := lookup(parser, rawIP)
		if result.Error != "" {
			hasError = true
		}
		results = append(results, result)
	}

	if err := writeResults(stdout, cfg.format, results); err != nil {
		return err
	}
	if hasError && cfg.failOnError {
		return errors.New("one or more queries failed")
	}
	return nil
}

func lookup(parser *iploc.QQWryParser, rawIP string) queryResult {
	result := queryResult{IP: rawIP}
	ip := net.ParseIP(rawIP)
	if ip == nil || ip.To4() == nil {
		result.Error = iploc.ErrInvalidIP.Error()
		return result
	}

	query, err := parser.QueryResult(ip)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.StartIP = query.StartIP.String()
	result.StopIP = query.StopIP.String()
	result.RecordA = query.RecordA
	result.RecordB = query.RecordB
	return result
}

func readIPs(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	var ips []string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		ips = append(ips, fields...)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stdin failed: %w", err)
	}
	return ips, nil
}

func writeResults(writer io.Writer, format string, results []queryResult) error {
	switch format {
	case "text":
		for _, result := range results {
			if result.Error != "" {
				fmt.Fprintf(writer, "%s\tERROR\t%s\n", result.IP, result.Error)
				continue
			}
			if result.RecordB == "" {
				fmt.Fprintf(writer, "%s\t%s\n", result.IP, result.RecordA)
				continue
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", result.IP, result.RecordA, result.RecordB)
		}
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	case "jsonl":
		encoder := json.NewEncoder(writer)
		for _, result := range results {
			if err := encoder.Encode(result); err != nil {
				return err
			}
		}
	}
	return nil
}

func printUsage(writer io.Writer) {
	fmt.Fprintf(writer, "Usage: iploc -db /path/to/qqwry.dat [flags] <ipv4> [ipv4...]\n")
	fmt.Fprintf(writer, "       echo '8.8.8.8' | iploc -db /path/to/qqwry.dat --format jsonl\n\n")
	fmt.Fprintln(writer, "Flags:")
	fmt.Fprintln(writer, "  -db string")
	fmt.Fprintln(writer, "        path to qqwry.dat database (required)")
	fmt.Fprintln(writer, "  -format string")
	fmt.Fprintln(writer, "        output format: text, json, jsonl (default \"text\")")
	fmt.Fprintln(writer, "  -map")
	fmt.Fprintln(writer, "        preload records into a map before querying")
	fmt.Fprintln(writer, "  -fail-on-error")
	fmt.Fprintln(writer, "        exit with code 1 if any input IP fails")
	fmt.Fprintln(writer, "  -version")
	fmt.Fprintln(writer, "        print version and exit")
}

type usageError string

func (e usageError) Error() string {
	return string(e)
}
