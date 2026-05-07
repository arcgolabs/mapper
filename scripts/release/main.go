package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	version := flag.String("version", "", "release version, e.g. v1.2.3 or 1.2.3")
	push := flag.Bool("push", false, "push the release tag to origin")
	dryRun := flag.Bool("dry-run", false, "print actions without making any changes")
	flag.Parse()

	if err := runRelease(*version, *push, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "release failed: %v\n", err)
		os.Exit(1)
	}
}

func runRelease(versionArg string, push bool, dryRun bool) error {
	if err := ensureCleanWorkingTree(); err != nil {
		return err
	}

	version, tag, err := normalizeVersion(versionArg)
	if err != nil {
		return err
	}

	if err := run("go", "fmt", "./..."); err != nil {
		return err
	}
	if err := run("go", "test", "./..."); err != nil {
		return err
	}
	if err := run("go", "mod", "tidy"); err != nil {
		return err
	}
	if err := ensureCleanWorkingTree(); err != nil {
		fmt.Fprintf(os.Stdout, "warning: formatting or module tidy changed files. Please commit changes first.\n%s\n", err)
		return err
	}

	if tagExists(tag) {
		return fmt.Errorf("release tag already exists: %s", tag)
	}

	fmt.Printf("prepared release for version %s (tag: %s)\n", version, tag)

	if dryRun {
		fmt.Println("dry-run: release steps skipped")
		return nil
	}

	if err := run("git", "tag", "-a", tag, "-m", fmt.Sprintf("release %s", version)); err != nil {
		return err
	}
	if push {
		if err := run("git", "push", "origin", tag); err != nil {
			return err
		}
	}

	fmt.Printf("release tag created: %s\n", tag)
	return nil
}

func normalizeVersion(versionArg string) (string, string, error) {
	if versionArg == "" {
		return "", "", errors.New("missing --version")
	}

	version := strings.TrimSpace(versionArg)
	if version == "" {
		return "", "", errors.New("missing --version")
	}

	re := regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z\.-]+)?$`)
	if !re.MatchString(version) {
		return "", "", fmt.Errorf("invalid semantic version: %s", version)
	}

	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return version, tag, nil
}

func run(command string, args ...string) error {
	c := exec.Command(command, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Dir = repoRoot()
	return c.Run()
}

func output(command string, args ...string) (string, error) {
	c := exec.Command(command, args...)
	c.Dir = repoRoot()
	c.Stderr = os.Stderr
	bytes, err := c.Output()
	return strings.TrimSpace(string(bytes)), err
}

func tagExists(tag string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/tags/"+tag)
	cmd.Dir = repoRoot()
	return cmd.Run() == nil
}

func ensureCleanWorkingTree() error {
	out, err := output("git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("working tree is not clean:\n%s", out)
	}
	return nil
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
