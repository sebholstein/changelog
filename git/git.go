package git

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const gitCommitGrep = "^feat|^fix|BREAKING"
const gitCommitFormat = "%H%n%s%n%b%n==END=="
const MAX_SUBJECT_LENGTH = 80

var closeRegex = regexp.MustCompile(`\s*(?:Closes|Fixes|Resolves)\s#(\d+)`)
var allTypesRegex = regexp.MustCompile(`(?:Closes|Fixes|Resolves)\s((?:#\d+(?:\,\s)?)+)`)
var numberRegex = regexp.MustCompile(`\d+`)
var breakingRegex = regexp.MustCompile(`/BREAKING CHANGE:\s([\s\S]*)/ig`)
var commitPatternRegex = regexp.MustCompile(`^(\w*)(\(([\w\$\.\-\* ]*)\))?\: (.*)$`)

func GetLatestTag() (tag string, err error) {
	hashCmd, err := exec.Command("git", "rev-list", "--tags", "--max-count=1").Output()
	if err != nil {
		return getFirstCommit()
	}

	hash := string(hashCmd)
	// trim result to get the hash without line breaks
	hash = strings.Trim(hash, "\r\n")

	gitCmd, err := exec.Command("git", "describe", "--tags", hash).Output()
	tag = strings.Trim(string(gitCmd), "\r\n")
	return
}

func getFirstCommit() (firstCommitHash string, err error) {
	gitCmd, err := exec.Command("git", "log", "--format=\"%H\"", "--pretty=oneline", "--reverse").Output()

	if err != nil {
		return
	}

	output := string(gitCmd)
	lines := strings.Split(output, "\n")
	firstLineSections := strings.Split(lines[0], " ")
	firstCommitHash = strings.TrimSpace(firstLineSections[0])
	return
}

type Commit struct {
	Hash      string
	Subject   string
	Closes    []int
	Breaks    []string
	Body      string
	Type      string
	Component string
}

func GetChangelogCommits(from string, to string) (commits []*Commit, err error) {
	commits = make([]*Commit, 0)
	//commitRange := from + ".." + to
	gitCmd := exec.Command("git", "log", "--format=\""+gitCommitFormat+"\"", "--grep=\""+gitCommitGrep+"\"", "-E")
	output, err := gitCmd.Output()
	if err != nil {
		return
	}

	splittedOutput := strings.Split(string(output), "==END==")
	for _, commit := range splittedOutput {
		commits = parseCommit(commit, commits)
	}

	return
}

func parseCommit(commit string, commits []*Commit) []*Commit {
	if commit == "" {
		return commits
	}

	commit = strings.TrimPrefix(commit, "\n")
	lines := strings.Split(commit, "\n")

	if len(lines) < 3 {
		return commits
	}

	newCommit := &Commit{}
	newCommit.Hash = strings.Replace(lines[1], "\n", "", 0)
	newCommit.Subject = strings.Replace(lines[2], "\n", "", 0)
	newCommit.Closes = make([]int, 0)
	newCommit.Breaks = make([]string, 0)
	newCommit.Body = commit

	// remove all issue numbers in subject
	closeRegex.ReplaceAllFunc([]byte(newCommit.Subject), func(s []byte) []byte {
		no, err := strconv.Atoi(string(s))
		if err != nil {
			return s
		}

		newCommit.Closes = append(newCommit.Closes, no)
		return []byte("")
	})

	// find all closes/fixes numbers in commit message
	for _, line := range lines {
		matches := allTypesRegex.FindAllString(line, -1)
		for _, match := range matches {
			for _, msg := range strings.Split(match, ",") {
				if numberRegex.MatchString(msg) {
					submatches := numberRegex.FindStringSubmatch(msg)
					if len(submatches) > 0 {
						number, _ := strconv.Atoi(submatches[0])
						newCommit.Closes = append(newCommit.Closes, number)
					}
				}
			}
		}
	}

	// find all breaking changes
	if matches := breakingRegex.FindStringSubmatch(commit); len(matches) > 1 {
		newCommit.Breaks = append(newCommit.Breaks, matches[1]+"\n")
	}

	// find component, type and subject
	submatches := commitPatternRegex.FindStringSubmatch(newCommit.Subject)

	if len(submatches) < 5 {
		return commits
	}

	// shorten subject
	if utf8.RuneCountInString(submatches[4]) > 80 {
		submatches[4] = runeSubstr(submatches[4], 0, 80)
	}

	newCommit.Type = submatches[1]
	newCommit.Component = submatches[3]
	newCommit.Subject = submatches[4]

	return append(commits, newCommit)
}

func runeSubstr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}
