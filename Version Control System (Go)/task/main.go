package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	vcsDir := filepath.Join(currentDir, "vcs")

	err = os.MkdirAll(filepath.Join(vcsDir, "commits"), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	if len(args) == 0 || args[0] == "--help" {
		printHelp()
		return
	}

	command := strings.TrimPrefix(args[0], "--")

	switch command {
	case "config":
		handleConfig(vcsDir, args[1:])
	case "add":
		handleAdd(vcsDir, args[1:])
	case "commit":
		if len(args) < 2 {
			fmt.Println("Message was not passed.")
		} else {
			commitChanges(vcsDir, args[1])
		}
	case "log":
		showLog(vcsDir)
	case "checkout":
		if len(args) < 2 {
			fmt.Println("Commit id was not passed.")
			return
		}
		checkoutCommit(vcsDir, args[1])
	default:
		fmt.Printf("'%s' is not a SVCS command.\n", command)
	}
}

func printHelp() {
	fmt.Println("These are SVCS commands:")
	fmt.Println("config     Get and set a username.")
	fmt.Println("add        Add a file to the index.")
	fmt.Println("log        Show commit logs.")
	fmt.Println("commit     Save changes.")
	fmt.Println("checkout   Restore a file.")
}

func handleConfig(vcsDir string, args []string) {
	configFile := filepath.Join(vcsDir, "config.txt")

	if len(args) == 0 {
		data, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Println("Please, tell me who you are.")
		} else {
			fmt.Printf("The username is %s.\n", strings.TrimSpace(string(data)))
		}
	} else {
		username := args[0]
		err := os.WriteFile(configFile, []byte(username), 0644)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The username is %s.\n", username)
	}
}

func handleAdd(vcsDir string, args []string) {
	indexFile := filepath.Join(vcsDir, "index.txt")

	if len(args) == 0 {
		data, err := os.ReadFile(indexFile)
		if err != nil || len(data) == 0 {
			fmt.Println("Add a file to the index.")
		} else {
			fmt.Println("Tracked files:")
			fmt.Println(string(data))
		}
	} else {
		filename := args[0]
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Printf("Can't find '%s'.\n", filename)
		} else {
			f, err := os.OpenFile(indexFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(f)
			if _, err := f.WriteString(filename + "\n"); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("The file '%s' is tracked.\n", filename)
		}
	}
}

func commitChanges(vcsDir, message string) {
	configFile := filepath.Join(vcsDir, "config.txt")
	author, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("Please, set a username first with the 'config' command.")
		return
	}

	indexFile := filepath.Join(vcsDir, "index.txt")
	trackedFiles, err := os.ReadFile(indexFile)
	if err != nil || len(trackedFiles) == 0 {
		fmt.Println("No files are tracked.")
		return
	}

	commitID, hasChanges := generateCommitID(strings.Split(string(trackedFiles), "\n"))
	if !hasChanges {
		fmt.Println("Nothing to commit.")
		return
	}

	commitDir := filepath.Join(vcsDir, "commits", commitID)

	if _, err := os.Stat(commitDir); err == nil {
		fmt.Println("Nothing to commit.")
		return
	}

	err = os.Mkdir(commitDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range strings.Split(string(trackedFiles), "\n") {
		if file == "" {
			continue
		}
		copyFileToCommit(file, commitDir)
	}

	logFile := filepath.Join(vcsDir, "log.txt")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(f)

	logEntry := fmt.Sprintf("commit %s\nAuthor: %s\n%s\n\n", commitID, strings.TrimSpace(string(author)), message)
	_, err = f.WriteString(logEntry)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Changes are committed.")
}

func generateCommitID(files []string) (string, bool) {
	hasher := sha256.New()
	hasChanges := false

	for _, file := range files {
		if file == "" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		hasher.Write(data)

		hasChanges = true
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), hasChanges
}

func copyFileToCommit(src, commitDir string) {
	dest := filepath.Join(commitDir, filepath.Base(src))
	srcFile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer func(srcFile *os.File) {
		err := srcFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(srcFile)

	destFile, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer func(destFile *os.File) {
		err := destFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(destFile)

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
}

func showLog(vcsDir string) {
	logFile := filepath.Join(vcsDir, "log.txt")
	logData, err := os.ReadFile(logFile)
	if err != nil || len(logData) == 0 {
		fmt.Println("No commits yet.")
		return
	}

	logEntries := strings.Split(string(logData), "\n\n")
	for i := len(logEntries) - 2; i >= 0; i-- {
		fmt.Println(logEntries[i])
		if i != 0 {
			fmt.Println()
		}
	}
}

func checkoutCommit(vcsDir string, commitID string) {
	commitDir := filepath.Join(vcsDir, "commits", commitID)

	if _, err := os.Stat(commitDir); os.IsNotExist(err) {
		fmt.Println("Commit does not exist.")
		return
	}

	indexFile := filepath.Join(vcsDir, "index.txt")
	trackedFiles, err := os.ReadFile(indexFile)
	if err != nil || len(trackedFiles) == 0 {
		fmt.Println("No files are tracked.")
		return
	}

	for _, file := range strings.Split(string(trackedFiles), "\n") {
		if file == "" {
			continue
		}
		restoreFileFromCommit(file, commitDir)
	}

	fmt.Printf("Switched to commit %s.\n", commitID)
}

func restoreFileFromCommit(filename, commitDir string) {
	src := filepath.Join(commitDir, filepath.Base(filename))
	dest := filepath.Join(".", filepath.Base(filename))
	srcFile, err := os.Open(src)

	if err != nil {
		log.Fatal(err)
	}
	defer func(srcFile *os.File) {
		err := srcFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(srcFile)

	destFile, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer func(destFile *os.File) {
		err := destFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(destFile)

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
}
