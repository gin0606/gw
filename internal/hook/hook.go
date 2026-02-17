package hook

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Run executes a hook script if it exists.
// Hook's stdout and stderr are both written to the output writer.
// Returns nil if the hook file does not exist (success).
// Returns an error if the hook file exists but is not executable, or if the hook exits non-zero.
func Run(repoRoot, hookName, cwd, worktreePath, branch string, output io.Writer) error {
	hookPath := filepath.Join(repoRoot, ".gw", "hooks", hookName)

	info, err := os.Stat(hookPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("hook %q is not executable", hookName)
	}

	cmd := exec.Command(hookPath)
	cmd.Dir = cwd
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = append(os.Environ(),
		"GW_REPO_ROOT="+repoRoot,
		"GW_WORKTREE_PATH="+worktreePath,
		"GW_BRANCH="+branch,
	)

	return cmd.Run()
}
