package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/nightlyone/lockfile"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func xtrabackup(targetDir string, incrementalBasedir string) {
	arg := []string{"--backup", "--target-dir=" + targetDir, "--parallel=4", "--user=root"}
	if incrementalBasedir != "" {
		arg = append(arg, "--incremental-basedir="+incrementalBasedir)
	}
	cmd := exec.Command("xtrabackup", arg...)

	stderr, err := cmd.StderrPipe()
	// stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(targetDir)
		panic(err)
	}

	if err = cmd.Start(); err != nil {
		_ = os.RemoveAll(targetDir)
		panic(err)
	}

	// print the output of the subprocess
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}

	if err = cmd.Wait(); err != nil {
		_ = os.RemoveAll(targetDir)
		panic(err)
	}

	// Fix file and dir permissions: All files and dirs readable
	if output, err := exec.Command("chmod", "-R", "+r", targetDir).CombinedOutput(); err != nil {
		fmt.Println(string(output))
		panic(err)
	}

	if output, err := exec.Command("sh", "-c", "find "+targetDir+" -type d -exec chmod +x {} +").CombinedOutput(); err != nil {
		fmt.Println(string(output))
		panic(err)
	}
}

var silentLock bool

func init() {
	flag.BoolVar(&silentLock, "silent-lock", false, "")
}

func main() {
	flag.Parse()

	if _, err := os.Stat(filepath.Join("/", "var", "tmp", "easybackup.disabled")); err == nil {
		fmt.Println("easybackup is manually disabled.")
		return
	}

	lock, err := lockfile.New(filepath.Join(os.TempDir(), "easybackup.pid"))
	if err != nil {
		panic(err)
	}

	if err = lock.TryLock(); err != nil {
		if silentLock {
			return
		}
		panic(err)
	}

	defer func() {
		if err := lock.Unlock(); err != nil {
			panic(err)
		}
	}()

	var backupDir string
	if len(flag.Args()) > 0 {
		backupDir = flag.Args()[0]
		if _, err := os.Stat(backupDir); os.IsPermission(err) || os.IsNotExist(err) {
			panic(err)
		}
	} else {
		backupDir = filepath.Join(os.TempDir(), "mysql")
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			if err := os.Mkdir(backupDir, os.ModePerm); err != nil {
				panic(err)
			}
		}
	}

	currentTime := time.Now()

	backupDirToday := filepath.Join(backupDir, currentTime.Format("2006-01-02"))
	if _, err := os.Stat(backupDirToday); os.IsNotExist(err) {
		if err := os.Mkdir(backupDirToday, os.ModePerm); err != nil {
			panic(err)
		}
	}

	backupDirBase := filepath.Join(backupDirToday, "base")
	if _, err := os.Stat(backupDirBase); os.IsNotExist(err) {
		xtrabackup(backupDirBase, "")

		backupDirYesterday := filepath.Join(backupDir, currentTime.AddDate(0, 0, -1).Format("2006-01-02"))
		if _, err := os.Stat(backupDirYesterday); !os.IsNotExist(err) {
			_ = os.RemoveAll(backupDirYesterday)
		}

		return
	}

	backupDirInc := filepath.Join(backupDirToday, "inc")
	if _, err := os.Stat(backupDirInc); os.IsNotExist(err) {
		if err := os.Mkdir(backupDirInc, os.ModePerm); err != nil {
			panic(err)
		}
	}

	files, _ := os.ReadDir(backupDirInc)
	incCount := len(files)
	var incrementalBasedir string
	if incCount == 0 {
		incrementalBasedir = backupDirBase
	} else {
		incrementalBasedir = filepath.Join(backupDirInc, fmt.Sprintf("%02d", incCount))
	}

	xtrabackup(filepath.Join(backupDirInc, fmt.Sprintf("%02d", incCount+1)), incrementalBasedir)
}
