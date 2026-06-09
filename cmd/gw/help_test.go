package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/config"
	"github.com/gin0606/gw/internal/help"
	"github.com/gin0606/gw/internal/hook"
	"github.com/gin0606/gw/internal/testutil"
)

func TestHelp_NoArgs_ShowsRootOverview(t *testing.T) {
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help")
	requireOK(t, "gw help", exitCode, stderr)
	if !strings.Contains(stdout, "COMMANDS") {
		t.Errorf("help (no args) should show command list, got: %q", stdout)
	}
	for _, topic := range help.TopicNames() {
		if !strings.Contains(stdout, topic) {
			t.Errorf("root overview should list topic %q, got: %q", topic, stdout)
		}
	}
}

func TestHelp_NoArgs_MatchesFlagHelp(t *testing.T) {
	bare, bareErr, bareCode := runGw(t, t.TempDir(), "help")
	flag, flagErr, flagCode := runGw(t, t.TempDir(), "--help")
	requireOK(t, "gw help", bareCode, bareErr)
	requireOK(t, "gw --help", flagCode, flagErr)
	if bare != flag {
		t.Errorf("`gw help` and `gw --help` should match.\nhelp:\n%s\n--help:\n%s", bare, flag)
	}
}

func TestHelp_Alias_h(t *testing.T) {
	viaAlias, aliasErr, aliasCode := runGw(t, t.TempDir(), "h", "hooks")
	viaFull, fullErr, fullCode := runGw(t, t.TempDir(), "help", "hooks")
	requireOK(t, "gw h hooks", aliasCode, aliasErr)
	requireOK(t, "gw help hooks", fullCode, fullErr)
	if viaAlias != viaFull {
		t.Errorf("`gw h hooks` and `gw help hooks` should match")
	}
}

func TestHelp_Command_MatchesFlagHelp(t *testing.T) {
	for _, name := range userFacingSubcommands(t) {
		viaHelp, helpErr, helpCode := runGw(t, t.TempDir(), "help", name)
		viaFlag, flagErr, flagCode := runGw(t, t.TempDir(), name, "--help")
		requireOK(t, "gw help "+name, helpCode, helpErr)
		requireOK(t, "gw "+name+" --help", flagCode, flagErr)
		if viaHelp != viaFlag {
			t.Errorf("`gw help %s` and `gw %s --help` should match.\nhelp:\n%s\n--help:\n%s", name, name, viaHelp, viaFlag)
		}
	}
}

func TestHelp_HelpFlagItself(t *testing.T) {
	// `gw help --help` and `gw help -h` should show help for the help command,
	// matching the behaviour of every other subcommand. Anchor on the help
	// command's UsageText, which is unique to its own help page (the root and
	// other subcommands never emit this exact line).
	const helpUsageLine = "gw help [<command>|<topic>|all]"
	for _, arg := range []string{"--help", "-h"} {
		stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", arg)
		requireOK(t, "gw help "+arg, exitCode, stderr)
		if !strings.Contains(stdout, helpUsageLine) {
			t.Errorf("`gw help %s` should show help for the help command (expected to contain %q), got: %q", arg, helpUsageLine, stdout)
		}
	}
}

func TestHelp_Topic(t *testing.T) {
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", "hooks")
	requireOK(t, "gw help hooks", exitCode, stderr)
	for _, name := range hook.Names() {
		if !strings.Contains(stdout, name) {
			t.Errorf("`gw help hooks` should mention %q, got: %q", name, stdout)
		}
	}
	for _, env := range hook.EnvVars() {
		if !strings.Contains(stdout, env) {
			t.Errorf("`gw help hooks` should mention %q, got: %q", env, stdout)
		}
	}
}

func TestHelp_CompletionTopicShadowsAutoSubcommand(t *testing.T) {
	// urfave's EnableShellCompletion registers a `completion` subcommand whose
	// own help only documents shell-script generation. The `completion` topic
	// is the more comprehensive doc, so `gw help completion` must resolve to
	// the topic, not the subcommand.
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", "completion")
	requireOK(t, "gw help completion", exitCode, stderr)
	if !strings.Contains(stdout, "DYNAMIC COMPLETION") {
		t.Errorf("`gw help completion` should return the topic (which mentions DYNAMIC COMPLETION), got: %q", stdout)
	}
}

func TestHelp_UnknownTopic(t *testing.T) {
	_, stderr, exitCode := runGw(t, t.TempDir(), "help", "does-not-exist")
	if exitCode != 3 {
		t.Errorf("expected exit code 3 (matching urfave's `No help topic` convention), got %d", exitCode)
	}
	if !strings.Contains(stderr, "No help topic for 'does-not-exist'") {
		t.Errorf("stderr should match urfave's wording, got: %q", stderr)
	}
}

func TestUsageError_NoDuplicateMessage(t *testing.T) {
	// Anchor on the exact line shape (single body line, single trailing
	// newline). A body-only Count or HasPrefix check would still pass against
	// the failure mode where urfave's default HandleExitCoder Fprintln's the
	// silent cli.Exit("", 1) that handleUsageError returns, appending a second
	// "\n" and producing a stray blank line after "Incorrect Usage: ...".
	repo := testutil.NewTestRepo(t)
	_, stderr, exitCode := runGw(t, repo.Root, "add", "--from")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing flag value")
	}
	want := "Incorrect Usage: flag needs an argument: --from\n"
	if stderr != want {
		t.Errorf("stderr should be exactly %q, got %q", want, stderr)
	}
}

func TestHelp_UnknownTarget_MatchesUnknownSubcommand(t *testing.T) {
	// `gw help <X>` and `gw <X>` should return the same wording and the same
	// exit code when X is neither a topic nor a subcommand. Both go through
	// urfave's convention (exit 3, "No help topic for 'X'"). Pin the output
	// contract to exactly one body occurrence per entry point so a future
	// change that adds a second print path (e.g. main() printing on top of
	// urfave's HandleExitCoder, or vice versa) is caught.
	const body = "No help topic for 'does-not-exist'"
	_, stderrHelp, codeHelp := runGw(t, t.TempDir(), "help", "does-not-exist")
	_, stderrDirect, codeDirect := runGw(t, t.TempDir(), "does-not-exist")
	if codeHelp != codeDirect {
		t.Errorf("`gw help X` and `gw X` should share an exit code, got %d vs %d", codeHelp, codeDirect)
	}
	for label, s := range map[string]string{"help X": stderrHelp, "direct X": stderrDirect} {
		if got := strings.Count(s, body); got != 1 {
			t.Errorf("%s: expected %q exactly once, got %d:\n%s", label, body, got, s)
		}
	}
}

func TestHelp_TooManyArgs(t *testing.T) {
	_, stderr, exitCode := runGw(t, t.TempDir(), "help", "hooks", "extra")
	if exitCode == 0 {
		t.Error("expected non-zero exit code for extra args")
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("stderr should report unexpected argument, got: %q", stderr)
	}
}

func TestHelp_All_ContainsAllSectionsAndIdentifiers(t *testing.T) {
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", "all")
	requireOK(t, "gw help all", exitCode, stderr)

	for _, h := range wantHelpAllHeaders(t) {
		if !strings.Contains(stdout, h) {
			t.Errorf("`gw help all` missing header %q", h)
		}
	}
	for _, name := range hook.Names() {
		if !strings.Contains(stdout, name) {
			t.Errorf("`gw help all` missing hook name %q", name)
		}
	}
	for _, env := range hook.EnvVars() {
		if !strings.Contains(stdout, env) {
			t.Errorf("`gw help all` missing env var %q", env)
		}
	}
	if !strings.Contains(stdout, config.KeyWorktreesDir) {
		t.Errorf("`gw help all` missing config key %q", config.KeyWorktreesDir)
	}
}

func TestHelp_All_SectionOrder(t *testing.T) {
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", "all")
	requireOK(t, "gw help all", exitCode, stderr)
	prev := -1
	for _, h := range wantHelpAllHeaders(t) {
		idx := strings.Index(stdout, h)
		if idx < 0 {
			t.Errorf("section %q not found", h)
			continue
		}
		if idx <= prev {
			t.Errorf("section %q appeared out of order (idx %d, prev %d)", h, idx, prev)
		}
		prev = idx
	}
}

func TestHelp_All_OverviewMatchesRootHelp(t *testing.T) {
	all, stderrAll, allCode := runGw(t, t.TempDir(), "help", "all")
	root, stderrRoot, rootCode := runGw(t, t.TempDir(), "--help")
	requireOK(t, "gw help all", allCode, stderrAll)
	requireOK(t, "gw --help", rootCode, stderrRoot)

	overviewStart := strings.Index(all, help.SectionHeader("OVERVIEW", ""))
	if overviewStart < 0 {
		t.Fatalf("OVERVIEW header missing from gw help all")
	}
	nextSection := strings.Index(all[overviewStart+1:], "===== ")
	if nextSection < 0 {
		t.Fatalf("could not locate next section after OVERVIEW")
	}
	overview := all[overviewStart : overviewStart+1+nextSection]
	if !strings.Contains(overview, strings.TrimSpace(root)) {
		t.Errorf("gw help all OVERVIEW section should embed `gw --help` output verbatim.\nOVERVIEW:\n%s\nROOT:\n%s", overview, root)
	}
}

func TestHelp_OutputContract_StdoutVsStderr(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"help", []string{"help"}},
		{"help_hooks", []string{"help", "hooks"}},
		{"help_add", []string{"help", "add"}},
		{"help_all", []string{"help", "all"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr, exitCode := runGw(t, t.TempDir(), c.args...)
			requireOK(t, "gw "+strings.Join(c.args, " "), exitCode, stderr)
			if stdout == "" {
				t.Errorf("expected non-empty stdout")
			}
			if stderr != "" {
				t.Errorf("expected empty stderr, got: %q", stderr)
			}
		})
	}
}

func TestHelp_UnknownTopic_ErrorOnStderrOnly(t *testing.T) {
	stdout, stderr, exitCode := runGw(t, t.TempDir(), "help", "does-not-exist")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout != "" {
		t.Errorf("expected empty stdout on error, got: %q", stdout)
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestHelpAllCommands_ExhaustsUserFacingSubcommands(t *testing.T) {
	// Catch silent omission: every user-visible subcommand of the root command
	// must appear in `gw help all`. Framework-managed commands (completion, help)
	// are explicitly excluded.
	stdout, _, _ := runGw(t, t.TempDir(), "help", "all")
	for _, name := range userFacingSubcommands(t) {
		want := help.SectionHeader("COMMAND", name)
		if !strings.Contains(stdout, want) {
			t.Errorf("`gw help all` missing COMMAND section for user-visible subcommand %q", name)
		}
	}
}

func TestInit_HookTemplatesReferenceHelpHooks(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	if _, stderr, exitCode := runGw(t, repo.Root, "init"); exitCode != 0 {
		t.Fatalf("gw init exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	for _, name := range hook.Names() {
		path := filepath.Join(repo.Root, ".gw", "hooks", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("stat %s: %v", name, err)
			continue
		}
		if info.Mode().Perm() != 0755 {
			t.Errorf("template %s should be mode 0755, got %v", name, info.Mode().Perm())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		body := string(data)
		if !strings.HasPrefix(body, "#!/bin/sh\n") {
			t.Errorf("template %s should start with #!/bin/sh, got: %q", name, body[:min(len(body), 40)])
		}
		if !strings.Contains(body, "gw help hooks") {
			t.Errorf("template %s should reference `gw help hooks`", name)
		}
		for _, env := range hook.EnvVars() {
			if !strings.Contains(body, env) {
				t.Errorf("template %s should mention %q", name, env)
			}
		}
		if err := exec.Command("sh", "-n", path).Run(); err != nil {
			t.Errorf("template %s failed `sh -n` syntax check: %v", name, err)
		}
	}
}

// wantHelpAllHeaders returns the canonical section headers expected in
// `gw help all`, derived from production helpers and the same root command
// inspection used by the implementation, so tests track production changes
// automatically.
func wantHelpAllHeaders(t *testing.T) []string {
	t.Helper()
	headers := []string{help.SectionHeader("OVERVIEW", "")}
	for _, name := range userFacingSubcommands(t) {
		headers = append(headers, help.SectionHeader("COMMAND", name))
	}
	for _, name := range help.TopicNames() {
		headers = append(headers, help.SectionHeader("TOPIC", name))
	}
	return headers
}

func userFacingSubcommands(t *testing.T) []string {
	t.Helper()
	// Parse the COMMANDS section of `gw --help`. Hidden commands (the
	// auto-injected `completion` subcommand) are absent from this list. We
	// also drop "help" itself since `gw help all` deliberately documents
	// commands but not the help command itself.
	stdout, _, _ := runGw(t, t.TempDir(), "--help")
	const marker = "COMMANDS:"
	idx := strings.Index(stdout, marker)
	if idx < 0 {
		t.Fatalf("COMMANDS section not found in gw --help output")
	}
	rest := stdout[idx+len(marker):]
	if end := strings.Index(rest, "\n\n"); end >= 0 {
		rest = rest[:end]
	}
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line is "<name>[, <alias>...]  <usage>".
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ",")
		if name == "help" || name == "h" {
			continue
		}
		out = append(out, name)
	}
	return out
}

func requireOK(t *testing.T, what string, exitCode int, stderr string) {
	t.Helper()
	if exitCode != 0 {
		t.Fatalf("%s: exit code = %d, want 0; stderr: %s", what, exitCode, stderr)
	}
}
