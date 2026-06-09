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

// Run executes a hook script if it exists. The hook's working directory is
// derived from name (see Name.cwd), so callers only supply repoRoot and the
// worktree path; the hook-vs-cwd mapping is not duplicated at every call
// site.
// Hook's stdout and stderr are both written to the output writer.
// Returns nil if the hook file does not exist (success).
// Returns an error if the hook file exists but is not executable, or if the hook exits non-zero.
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
		return fmt.Errorf("hook %q is not executable", name)
	}

	cmd := exec.Command(hookPath)
	cmd.Dir = name.cwd(repoRoot, worktreePath)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = append(os.Environ(),
		EnvRepoRoot+"="+repoRoot,
		EnvWorktreePath+"="+worktreePath,
		EnvBranch+"="+branch,
	)

	return cmd.Run()
}
