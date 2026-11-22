// Command sync-upstream checks patchright-python / patchright-core for new
// releases and applies the parts that map onto this Go port (driver version
// bump plus a snapshot of the Python patcher).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	pythonRepo     = "Kaliiiiiiiiii-Vinyzu/patchright-python"
	userAgent      = "patchright-go-sync"
	lockRelPath    = "upstream/lock.json"
	patcherRelPath = "upstream/patch_python_package.py"
	pendingRelPath = "upstream/PENDING.md"
	runGoRelPath   = "run.go"
)

type lockFile struct {
	PatchedAt           string `json:"patched_at"`
	PatchrightPythonSHA string `json:"patchright_python_sha"`
	PatchrightPyPI      string `json:"patchright_pypi"`
	PatchrightCore      string `json:"patchright_core"`
	PatcherSHA256       string `json:"patcher_sha256"`
}

type syncResult struct {
	Changed        bool
	DriverBumped   bool
	PatcherChanged bool
	Summary        string
	Previous       lockFile
	Next           lockFile
}

func main() {
	root := flag.String("repo-root", ".", "patchright-go repository root")
	flag.Parse()
	res, err := syncUpstream(*root, http.DefaultClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync-upstream: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(res.Summary)
	if p := os.Getenv("GITHUB_OUTPUT"); p != "" {
		if err := appendGitHubOutput(p, res); err != nil {
			fmt.Fprintf(os.Stderr, "github output: %v\n", err)
			os.Exit(1)
		}
	}
	if res.Changed {
		os.Exit(0)
	}
}

func appendGitHubOutput(path string, res syncResult) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	changed := "false"
	if res.Changed {
		changed = "true"
	}
	pending := "false"
	if res.PatcherChanged {
		pending = "true"
	}
	_, err = fmt.Fprintf(f, "changed=%s\npending_patcher=%s\n", changed, pending)
	return err
}

func syncUpstream(root string, client *http.Client) (syncResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return syncResult{}, err
	}
	prev, err := loadLock(filepath.Join(root, lockRelPath))
	if err != nil {
		return syncResult{}, err
	}
	core, err := fetchJSON(client, "https://registry.npmjs.org/patchright-core/latest", struct {
		Version string `json:"version"`
	}{})
	if err != nil {
		return syncResult{}, fmt.Errorf("patchright-core: %w", err)
	}
	pypi, err := fetchJSON(client, "https://pypi.org/pypi/patchright/json", struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}{})
	if err != nil {
		return syncResult{}, fmt.Errorf("pypi: %w", err)
	}
	commit, err := fetchJSON(client, "https://api.github.com/repos/"+pythonRepo+"/commits/main", struct {
		SHA string `json:"sha"`
	}{})
	if err != nil {
		return syncResult{}, fmt.Errorf("python repo: %w", err)
	}
	patcher, err := fetchBytes(client, "https://raw.githubusercontent.com/"+pythonRepo+"/main/patch_python_package.py")
	if err != nil {
		return syncResult{}, fmt.Errorf("python patcher: %w", err)
	}
	patcherSum := sha256Hex(patcher)

	next := lockFile{
		PatchedAt:           time.Now().UTC().Format(time.RFC3339),
		PatchrightPythonSHA: commit.SHA,
		PatchrightPyPI:      pypi.Info.Version,
		PatchrightCore:      core.Version,
		PatcherSHA256:       patcherSum,
	}

	res := syncResult{Previous: prev, Next: next}
	var notes []string

	runGoPath := filepath.Join(root, runGoRelPath)
	runGo, err := os.ReadFile(runGoPath)
	if err != nil {
		return syncResult{}, err
	}
	currentDriver, err := currentCLIVersion(string(runGo))
	if err != nil {
		return syncResult{}, err
	}
	if cmpVersion(next.PatchrightCore, currentDriver) > 0 {
		updated, err := bumpPlaywrightCliVersion(string(runGo), next.PatchrightCore)
		if err != nil {
			return syncResult{}, err
		}
		if err := os.WriteFile(runGoPath, []byte(updated), 0o644); err != nil {
			return syncResult{}, err
		}
		res.DriverBumped = true
		res.Changed = true
		notes = append(notes, fmt.Sprintf("driver %s -> %s", currentDriver, next.PatchrightCore))
	} else {
		next.PatchrightCore = currentDriver
	}

	if err := os.MkdirAll(filepath.Join(root, "upstream"), 0o755); err != nil {
		return syncResult{}, err
	}
	oldPatcher, _ := os.ReadFile(filepath.Join(root, patcherRelPath))
	if patcherSum != prev.PatcherSHA256 || string(oldPatcher) != string(patcher) {
		if err := os.WriteFile(filepath.Join(root, patcherRelPath), patcher, 0o644); err != nil {
			return syncResult{}, err
		}
		if prev.PatcherSHA256 != "" && prev.PatcherSHA256 != patcherSum {
			res.PatcherChanged = true
			res.Changed = true
			if err := writePending(root, prev, next); err != nil {
				return syncResult{}, err
			}
			notes = append(notes, "python patcher changed (see upstream/PENDING.md)")
		} else if prev.PatcherSHA256 == "" {
			res.Changed = true
			notes = append(notes, "snapshot python patcher")
		}
	}

	if next.PatchrightPythonSHA != prev.PatchrightPythonSHA || next.PatchrightPyPI != prev.PatchrightPyPI {
		res.Changed = true
		if next.PatchrightPyPI != prev.PatchrightPyPI && prev.PatchrightPyPI != "" {
			notes = append(notes, fmt.Sprintf("pypi %s -> %s", prev.PatchrightPyPI, next.PatchrightPyPI))
		}
		if next.PatchrightPythonSHA != prev.PatchrightPythonSHA && prev.PatchrightPythonSHA != "" {
			notes = append(notes, "python main "+shortSHA(next.PatchrightPythonSHA))
		}
	}

	if !res.PatcherChanged {
		_ = os.Remove(filepath.Join(root, pendingRelPath))
	}

	if res.Changed {
		if err := writeLock(filepath.Join(root, lockRelPath), next); err != nil {
			return syncResult{}, err
		}
	}

	if len(notes) == 0 {
		res.Summary = fmt.Sprintf("up to date (core %s, pypi %s, python %s)", next.PatchrightCore, next.PatchrightPyPI, shortSHA(next.PatchrightPythonSHA))
	} else {
		res.Summary = "applied: " + strings.Join(notes, "; ")
	}
	return res, nil
}

func loadLock(path string) (lockFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return lockFile{}, nil
	}
	if err != nil {
		return lockFile{}, err
	}
	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return lockFile{}, err
	}
	return lock, nil
}

func writeLock(path string, lock lockFile) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writePending(root string, prev, next lockFile) error {
	body := fmt.Sprintf(`# Pending Python patcher port

patchright-python main moved to %s (PyPI %s).
The Python AST patcher changed (%s -> %s). Driver bumps are applied automatically; port any new client-side patches from `+"`upstream/patch_python_package.py`"+` into this Go module.

Previous python SHA: %s
`, next.PatchrightPythonSHA, next.PatchrightPyPI, shortSHA(prev.PatcherSHA256), shortSHA(next.PatcherSHA256), prev.PatchrightPythonSHA)
	return os.WriteFile(filepath.Join(root, pendingRelPath), []byte(body), 0o644)
}

var cliVersionRe = regexp.MustCompile(`playwrightCliVersion = "([^"]+)"`)

func currentCLIVersion(runGo string) (string, error) {
	m := cliVersionRe.FindStringSubmatch(runGo)
	if len(m) != 2 {
		return "", fmt.Errorf("playwrightCliVersion not found in run.go")
	}
	return m[1], nil
}

func bumpPlaywrightCliVersion(runGo, version string) (string, error) {
	if !cliVersionRe.MatchString(runGo) {
		return "", fmt.Errorf("playwrightCliVersion not found in run.go")
	}
	return cliVersionRe.ReplaceAllString(runGo, `playwrightCliVersion = "`+version+`"`), nil
}

func cmpVersion(a, b string) int {
	as := parseVersion(a)
	bs := parseVersion(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func fetchBytes(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return body, nil
}

func fetchJSON[T any](client *http.Client, url string, dest T) (T, error) {
	body, err := fetchBytes(client, url)
	if err != nil {
		return dest, err
	}
	if err := json.Unmarshal(body, &dest); err != nil {
		return dest, err
	}
	return dest, nil
}
