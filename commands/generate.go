// Changelog
//
// Copyright (c) 2014 Sebastian Müller <info@sebastian-mueller.net>
//
// https://github.com/SebastianM/changelog
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package commands

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/sebastianm/changelog/generator"
	"github.com/sebastianm/changelog/git"
	"io/ioutil"
	"os"
)

const generateCommandName = "generate"

var Generate = cli.Command{
	Name:   generateCommandName,
	Usage:  "generate the changelog for a version",
	Action: generate,
	Flags: []cli.Flag{
		cli.StringFlag{"version, v", "", "Required. The version to be written to the changelog"},
		cli.StringFlag{"file, f", "CHANGELOG.md", "Which file to read the current changelog from and prepend the new changelog's contents to"},
		cli.StringFlag{"repository, r", "", "If this is provided, allows issues and commit hashes to be linked to the actual commit. Usually used with github repositories"},
		cli.StringFlag{"start, s", "", "Which commit the changelog should start at. By default, uses previous tag, or if no previous tag the first commit"},
		cli.StringFlag{"end, e", "HEAD", "Which commit the changelog should end at. By default, uses HEAD"},
	},
}

func generate(c *cli.Context) {
	checkNotEmptyStringFlag(c, "version", "No version provided", generateCommandName)

	from, to := c.String("from"), c.String("to")

	if from == "" {
		tag, err := git.GetLatestTag()
		if err != nil {
			errorExit("failed to read git tags: ", err.Error())
		}
		from = tag
	}

	if to == "" {
		to = "HEAD"
	}

	fmt.Printf("Generating changelog from %s to %s...\n", from, to)

	commits, _ := git.GetChangelogCommits(from, to)
	writeChangelog(c.String("file"), commits)
}

func writeChangelog(filename string, commits []*git.Commit, c *cli.Context) {
	fmt.Printf("Parsed %d commits\n", len(commits))

	_, err := os.Stat(filename)

	if err != nil && !os.IsNotExist(err) {
		fmt.Println("ERROR getting file stats: " + filename + " - " + err.Error())
		os.Exit(1)
	}

	var existingContent string

	// create changelog file if not exists
	if os.IsNotExist(err) {
		_, err = os.Create(filename)

		if err != nil {
			fmt.Println("ERROR creating file " + filename + " - " + err.Error())
			os.Exit(1)
		}
	} else {
		// grab existing content if file exists
		contentBytes, err := ioutil.ReadFile(filename)

		if err != nil {
			fmt.Println("ERROR creating file " + filename + " - " + err.Error())
			os.Exit(1)
		}

		existingContent = string(contentBytes)
	}

	newContent, err := generator.GenerateNewChangelogContent(existingContent, commits, c.String("version"))
	err = ioutil.WriteFile(filename, []byte(newContent), 0644)

	if err != nil {
		fmt.Println("ERROR writing new file content " + filename + " - " + err.Error())
		os.Exit(1)
	}
}
