package main

import (
	"bufio"
	"fmt"
	"github.com/nightlyone/lockfile"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func xtrabackup(targetDir string, incrementalBasedir string) {
	arg := []string{"--backup", "--target-dir=" + targetDir, "--parallel=4"}
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
}

func main() {

	lock, err := lockfile.New(filepath.Join(os.TempDir(), "easybackup"))
	if err != nil {
		panic(err)
	}

	if err = lock.TryLock(); err != nil {
		panic(err)
	}

	defer func() {
		if err := lock.Unlock(); err != nil {
			panic(err)
		}
	}()

	var backupDir string
	if os.Args != nil && len(os.Args) > 1 {
		backupDir = os.Args[1]
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

	backupDirToday := backupDir + "/" + currentTime.Format("2006-01-02")
	if _, err := os.Stat(backupDirToday); os.IsNotExist(err) {
		if err := os.Mkdir(backupDirToday, os.ModePerm); err != nil {
			panic(err)
		}
	}

	backupDirBase := backupDirToday + "/base"
	if _, err := os.Stat(backupDirBase); os.IsNotExist(err) {
		xtrabackup(backupDirBase, "")

		backupDirYesterday := backupDir + "/" + currentTime.AddDate(0, 0, -1).Format("2006-01-02")
		if _, err := os.Stat(backupDirYesterday); !os.IsNotExist(err) {
			_ = os.RemoveAll(backupDirYesterday)
		}

		return
	}

	backupDirInc := backupDirToday + "/inc"
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
		incrementalBasedir = filepath.Join(backupDirInc, strconv.Itoa(incCount))
	}

	xtrabackup(filepath.Join(backupDirInc, strconv.Itoa(incCount+1)), incrementalBasedir)
}
