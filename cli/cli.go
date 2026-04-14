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

	"github.com/5000K/5000mails/api"
)

const usageHeader = `Usage: 5kmcli [global flags] <command> [subcommand] [flags]

Global flags:
  --server URL             Server base URL (default: http://localhost:9000)
  --private-key-path PATH  Path to Ed25519 private key file for authentication

Commands:
  list   create   --name NAME                       Create a mailing list
  list   get      --id ID                           Get list details and stats
  list   rename   --id ID --name NAME               Rename a mailing list
  list   delete   --id ID                           Delete a mailing list
  list   users    --id ID                           List subscribers

  send   list     --list NAME --raw-path PATH       Send mail to all confirmed subscribers
  send   test     --name NAME --email EMAIL         Send a test mail
                  --raw-path PATH

  keys   generate [--out-dir DIR]                   Generate an Ed25519 key pair

Options for send commands:
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
		fmt.Fprintln(stderr, "usage: 5kmcli list <create|get|rename|delete|users> [flags]")
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
	id, ok := flagUint(args, "--id")
	if !ok {
		fmt.Fprintln(stderr, "usage: 5kmcli list get --id ID")
		return 1
	}
	resp, err := client.GetList(context.Background(), id)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listRename(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	id, ok := flagUint(args, "--id")
	name := flagValue(args, "--name")
	if !ok || name == "" {
		fmt.Fprintln(stderr, "usage: 5kmcli list rename --id ID --name NAME")
		return 1
	}
	resp, err := client.RenameList(context.Background(), id, name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printJSON(stdout, resp)
	return 0
}

func listDelete(args []string, client *api.PrivateClient, _, stderr io.Writer) int {
	id, ok := flagUint(args, "--id")
	if !ok {
		fmt.Fprintln(stderr, "usage: 5kmcli list delete --id ID")
		return 1
	}
	if err := client.DeleteList(context.Background(), id); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func listUsers(args []string, client *api.PrivateClient, stdout, stderr io.Writer) int {
	id, ok := flagUint(args, "--id")
	if !ok {
		fmt.Fprintln(stderr, "usage: 5kmcli list users --id ID")
		return 1
	}
	resp, err := client.GetUsers(context.Background(), id)
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
		fmt.Fprintln(stderr, "usage: 5kmcli send list --list NAME --raw-path PATH [--data KEY=VALUE ...]")
		return 1
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Fprintf(stderr, "error reading raw file: %v\n", err)
		return 1
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

func flagUint(args []string, name string) (uint, bool) {
	v := flagValue(args, name)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
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
