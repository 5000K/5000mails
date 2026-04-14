package cli

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/5000K/5000mails/api"
)

const usageHeader = `Usage: 5kmcli [global flags] <command> [subcommand] [flags]

Global flags:
  --server URL             Server base URL (default: http://localhost:9000)
  --private-key-path PATH  Path to Ed25519 private key file for authentication

Commands:
  list     all                                         List all mailing lists
  list     create   --name NAME                       Create a mailing list
  list     get      --name NAME                       Get list details and stats
  list     rename   --name NAME --new-name NEWNAME    Rename a mailing list
  list     delete   --name NAME                       Delete a mailing list
  list     users    --name NAME                       List subscribers

  send     list     --list NAME --raw-path PATH       Send mail immediately
                    [--at ISO8601] [--timezone TZ]    Schedule instead of sending immediately
  send     test     --email EMAIL --raw-path PATH     Send a test mail
                    [--name NAME]

  schedule list                                        List all scheduled mails
  schedule get      --id ID                           Get a scheduled mail
  schedule delete   --id ID                           Delete a scheduled mail
  schedule reschedule --id ID --at ISO8601            Reschedule a mail
                    [--timezone TZ]
  schedule content  --id ID --raw-path PATH           Replace content of a scheduled mail

  keys     generate [--out-dir DIR]                   Generate an Ed25519 key pair

Time flags:
  --at ISO8601       ISO 8601 datetime, assumed UTC (e.g. 2026-04-15T10:00:00)
  --timezone TZ      IANA timezone name overriding the UTC default (e.g. Europe/Berlin)

Options for send/schedule commands:
  --data KEY=VALUE   Template variable (repeatable)
`

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprint(stderr, usageHeader)
		return 1
	}

	var serverURL string
	var keyPath string
	remaining := parseGlobalFlags(args, &serverURL, &keyPath)

	if serverURL == "" {
		serverURL = "http://localhost:9000"
	}

	if len(remaining) == 0 {
		fmt.Fprint(stderr, usageHeader)
		return 1
	}

	command := remaining[0]
	rest := remaining[1:]

	switch command {
	case "list":
		return runList(rest, serverURL, keyPath, stdout, stderr)
	case "send":
		return runSend(rest, serverURL, keyPath, stdout, stderr)
	case "schedule":
		return runSchedule(rest, serverURL, keyPath, stdout, stderr)
	case "keys":
		return runKeys(rest, stdout, stderr)
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usageHeader)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n%s", command, usageHeader)
		return 1
	}
}

func runList(args []string, serverURL, keyPath string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: 5kmcli list <all|create|get|rename|delete|users> [flags]")
		return 1
	}

	client, err := buildClient(serverURL, keyPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sub := args[0]
	flags := args[1:]

	switch sub {
	case "all":
		return listAll(client, stdout, stderr)
	case "create":
		return listCreate(flags, client, stdout, stderr)
	case "get":
		return listGet(flags, client, stdout, stderr)
	case "rename":
		return listRename(flags, client, stdout, stderr)
	case "delete":
		return listDelete(flags, client, stdout, stderr)
	case "users":
		return listUsers(flags, client, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown list subcommand: %s\n", sub)
		return 1
	}
}

func runSend(args []string, serverURL, keyPath string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: 5kmcli send <list|test> [flags]")
		return 1
	}

	client, err := buildClient(serverURL, keyPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sub := args[0]
	flags := args[1:]

	switch sub {
	case "list":
		return sendList(flags, client, stdout, stderr)
	case "test":
		return sendTest(flags, client, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown send subcommand: %s\n", sub)
		return 1
	}
}

func runKeys(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: 5kmcli keys generate [--out-dir DIR]")
		return 1
	}
	if args[0] != "generate" {
		fmt.Fprintf(stderr, "unknown keys subcommand: %s\n", args[0])
		return 1
	}
	outDir := "."
	for i := 1; i < len(args)-1; i++ {
		if args[i] == "--out-dir" {
			outDir = args[i+1]
		}
	}
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(stderr, "error generating key pair: %v\n", err)
		return 1
	}
	pubPath, privPath, err := WriteKeyPair(outDir, pub, priv)
	if err != nil {
		fmt.Fprintf(stderr, "error writing key pair: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "private key: %s\npublic key:  %s\n", privPath, pubPath)
	return 0
}

func listAll(client *api.PrivateClient, stdout, stderr io.Writer) int {
	resp, err := client.GetAllLists(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listCreate(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	name := flagValue(args, "--name")
	if name == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list create --name NAME")
		return 1
	}
	resp, err := client.CreateList(context.Background(), name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listGet(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	name := flagValue(args, "--name")
	if name == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list get --name NAME")
		return 1
	}
	resp, err := client.GetList(context.Background(), name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listRename(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	name := flagValue(args, "--name")
	newName := flagValue(args, "--new-name")
	if name == "" || newName == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list rename --name NAME --new-name NEWNAME")
		return 1
	}
	resp, err := client.RenameList(context.Background(), name, newName)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listDelete(args []string, client *api.PrivateClient, _, stderr io.Writer) int {
	name := flagValue(args, "--name")
	if name == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list delete --name NAME")
		return 1
	}
	if err := client.DeleteList(context.Background(), name); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func listUsers(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	name := flagValue(args, "--name")
	if name == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list users --name NAME")
		return 1
	}
	resp, err := client.GetUsers(context.Background(), name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func sendList(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	listName := flagValue(args, "--list")
	rawPath := flagValue(args, "--raw-path")
	if listName == "" || rawPath == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli send list --list NAME --raw-path PATH [--at ISO8601] [--timezone TZ] [--data KEY=VALUE ...]")
		return 1
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Fprintf(stderr, "error reading raw file: %v\n", err)
		return 1
	}

	atStr := flagValue(args, "--at")
	if atStr != "" {
		scheduledAt, err := parseTimestamp(atStr, flagValue(args, "--timezone"))
		if err != nil {
			fmt.Fprintf(stderr, "error parsing --at: %v\n", err)
			return 1
		}
		m, err := client.ScheduleMail(context.Background(), listName, string(raw), scheduledAt)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		printJSON(stdout, m)
		return 0
	}

	data := collectData(args)
	if err := client.SendToList(context.Background(), listName, string(raw), data); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "mail dispatched")
	return 0
}

func sendTest(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	email := flagValue(args, "--email")
	rawPath := flagValue(args, "--raw-path")
	if email == "" || rawPath == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli send test --email EMAIL --raw-path PATH [--name NAME] [--data KEY=VALUE ...]")
		return 1
	}
	name := flagValue(args, "--name")
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Fprintf(stderr, "error reading raw file: %v\n", err)
		return 1
	}
	data := collectData(args)
	recipient := api.RecipientInput{Name: name, Email: email}
	if err := client.SendTestMail(context.Background(), recipient, string(raw), data); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "test mail sent")
	return 0
}

func buildClient(serverURL, keyPath string) (*api.PrivateClient, error) {
	var key ed25519.PrivateKey
	if keyPath != "" {
		k, err := ReadPrivateKey(keyPath)
		if err != nil {
			return nil, fmt.Errorf("loading private key: %w", err)
		}
		key = k
	}
	return api.NewPrivateClient(serverURL, key), nil
}

func parseGlobalFlags(args []string, serverURL, keyPath *string) []string {
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--server":
			if i+1 < len(args) {
				*serverURL = args[i+1]
				i++
			}
		case "--private-key-path":
			if i+1 < len(args) {
				*keyPath = args[i+1]
				i++
			}
		default:
			remaining = append(remaining, args[i:]...)
			return remaining
		}
	}
	return remaining
}

func flagValue(args []string, name string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func collectData(args []string) map[string]any {
	data := make(map[string]any)
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--data" {
			kv := args[i+1]
			if idx := strings.IndexByte(kv, '='); idx > 0 {
				data[kv[:idx]] = kv[idx+1:]
			}
		}
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func printJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// parseTimestamp converts an ISO 8601 datetime string to a unix timestamp.
// If tz is empty the input is assumed to be UTC.
func parseTimestamp(raw, tz string) (int64, error) {
	loc := time.UTC
	if tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return 0, fmt.Errorf("unknown timezone %q: %w", tz, err)
		}
	}

	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, raw, loc); err == nil {
			return t.Unix(), nil
		}
	}
	return 0, fmt.Errorf("cannot parse %q as ISO 8601 datetime", raw)
}

func runSchedule(args []string, serverURL, keyPath string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: 5kmcli schedule <list|get|delete|reschedule|content> [flags]")
		return 1
	}

	client, err := buildClient(serverURL, keyPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sub := args[0]
	flags := args[1:]

	switch sub {
	case "list":
		return scheduleList(client, stdout, stderr)
	case "get":
		return scheduleGet(flags, client, stdout, stderr)
	case "delete":
		return scheduleDelete(flags, client, stderr)
	case "reschedule":
		return scheduleReschedule(flags, client, stdout, stderr)
	case "content":
		return scheduleContent(flags, client, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown schedule subcommand: %s\n", sub)
		return 1
	}
}

func scheduleList(client *api.PrivateClient, stdout, stderr io.Writer) int {
	mails, err := client.GetAllScheduled(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, mails)
	return 0
}

func scheduleGet(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	idStr := flagValue(args, "--id")
	if idStr == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli schedule get --id ID")
		return 1
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fmt.Fprintf(stderr, "invalid id: %v\n", err)
		return 1
	}
	m, err := client.GetScheduled(context.Background(), uint(id))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, m)
	return 0
}

func scheduleDelete(args []string, client *api.PrivateClient, stderr io.Writer) int {
	idStr := flagValue(args, "--id")
	if idStr == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli schedule delete --id ID")
		return 1
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fmt.Fprintf(stderr, "invalid id: %v\n", err)
		return 1
	}
	if err := client.DeleteScheduled(context.Background(), uint(id)); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func scheduleReschedule(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	idStr := flagValue(args, "--id")
	atStr := flagValue(args, "--at")
	if idStr == "" || atStr == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli schedule reschedule --id ID --at ISO8601 [--timezone TZ]")
		return 1
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fmt.Fprintf(stderr, "invalid id: %v\n", err)
		return 1
	}
	scheduledAt, err := parseTimestamp(atStr, flagValue(args, "--timezone"))
	if err != nil {
		fmt.Fprintf(stderr, "error parsing --at: %v\n", err)
		return 1
	}
	m, err := client.RescheduleMail(context.Background(), uint(id), scheduledAt)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, m)
	return 0
}

func scheduleContent(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	idStr := flagValue(args, "--id")
	rawPath := flagValue(args, "--raw-path")
	if idStr == "" || rawPath == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli schedule content --id ID --raw-path PATH")
		return 1
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fmt.Fprintf(stderr, "invalid id: %v\n", err)
		return 1
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Fprintf(stderr, "error reading raw file: %v\n", err)
		return 1
	}
	m, err := client.ReplaceScheduledContent(context.Background(), uint(id), string(raw))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, m)
	return 0
}
