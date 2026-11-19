package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

func main() {
	changed := flag.Bool("changed", false, "classify the working tree's changed and untracked files")
	list := flag.Bool("list", false, "print every area and its path patterns")
	check := flag.Bool("check", false, "exit non-zero when the files span more than one area")
	unmatched := flag.Bool("unmatched", false, "print only paths no rule claimed")
	stdin := flag.Bool("stdin", false, "read one path per line from standard input")
	flag.Parse()

	if *list {
		printAreas()
		return
	}

	files := flag.Args()
	if *stdin {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); line != "" {
				files = append(files, line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "branchmap:", err)
			os.Exit(2)
		}
	}
	if *changed {
		var err error
		files, err = changedFiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, "branchmap:", err)
			os.Exit(2)
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "usage: branchmap [--changed] [--check] [--unmatched] <files...>")
		os.Exit(2)
	}

	seen := map[string]bool{}
	unmatchedFiles := 0
	for _, file := range files {
		file = normalize(file)
		if file == "" {
			continue
		}
		area, ok := Classify(file)
		if !ok {
			unmatchedFiles++
			if *unmatched {
				fmt.Printf("unmatched\t%s\n", file)
				continue
			}
			fmt.Fprintf(os.Stderr, "branchmap: %s has no rule; assigned to %s\n", file, area.Prefix)
		}
		if *unmatched {
			continue
		}
		fmt.Printf("%s\t%s\n", area.Prefix, file)
		seen[area.Prefix] = true
	}

	if *check {
		if len(seen) > 1 {
			branches := make([]string, 0, len(seen))
			for branch := range seen {
				branches = append(branches, branch)
			}
			sort.Strings(branches)
			fmt.Fprintf(os.Stderr, "branchmap: files span %d areas: %s\n", len(branches), strings.Join(branches, ", "))
			os.Exit(1)
		}
	}
	if *unmatched && unmatchedFiles == 0 {
		return
	}
}

func printAreas() {
	for _, area := range areas {
		fmt.Printf("%s  %s\n", area.Prefix, area.Note)
		for _, pattern := range area.Patterns {
			fmt.Printf("    %s\n", pattern)
		}
	}
}

func changedFiles() ([]string, error) {
	tracked, err := gitLines("diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	untracked, err := gitLines("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	files := make([]string, 0, len(tracked)+len(untracked))
	for _, file := range append(tracked, untracked...) {
		file = strings.TrimSpace(file)
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		files = append(files, file)
	}
	sort.Strings(files)
	return files, nil
}

func gitLines(args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}
