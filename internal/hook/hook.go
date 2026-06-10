package hook

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// Run executes the hook script at .gw/hooks/<name> with cwd derived from
// name (see Name.cwd). A missing hook file is a no-op; a hook that exists
// but lacks the executable bit returns an error rather than being silently
// skipped, so typos in hook filenames surface loudly. Hook stdout and
// stderr are both written to output.
func Run(repoRoot string, name Name, worktreePath, branch string, output io.Writer) error {
	hookPath := filepath.Join(repoRoot, ".gw", "hooks", string(name))

	info, err := os.Stat(hookPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat hook %q: %w", name, err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("hook %s is not executable (chmod +x)", hookPath)
	}

	cmd := exec.Command(hookPath)
	cmd.Dir = name.cwd(repoRoot, worktreePath)
	// Stdin is left unset so os/exec attaches the null device, matching the
	// "stdin reads EOF immediately" contract documented in `gw help hooks`.
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = append(os.Environ(),
		EnvRepoRoot+"="+repoRoot,
		EnvWorktreePath+"="+worktreePath,
		EnvBranch+"="+branch,
	)

	return cmd.Run()
}
