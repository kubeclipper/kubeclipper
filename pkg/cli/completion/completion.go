/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package completion

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

var (
	completionLong = `
  Output shell completion code for the specified shell (bash or zsh).
  The shell code must be evaluated to provide interactive
  completion of kcctl commands.  This can be done by sourcing it from
  the .bash_profile.

  Note for zsh users: [1] zsh completions are only supported in versions of zsh >= 5.2`

	completionExample = `
  # Installing bash completion on macOS using homebrew
  ## If running Bash 3.2 included with macOS
    brew install bash-completion
  ## or, if running Bash 4.1+
    brew install bash-completion@2
  ## If kcctl is installed via homebrew, this should start working immediately.
  ## If you've installed via other means, you may need add the completion to your completion directory
    kcctl completion bash > $(brew --prefix)/etc/bash_completion.d/kcctl


  # Installing bash completion on Linux
  ## If bash-completion is not installed on Linux, please install the 'bash-completion' package
  ## via your distribution's package manager.
  ## Load the kcctl completion code for bash into the current shell
    source <(kcctl completion bash)
  ## Write bash completion code to a file and source if from .bash_profile
    kcctl completion bash > ~/.kube/completion.bash.inc
    printf "
      # kcctl shell completion
      source '$HOME/.kube/completion.bash.inc'
      " >> $HOME/.bash_profile
    source $HOME/.bash_profile

  # Load the kcctl completion code for zsh[1] into the current shell
    source <(kcctl completion zsh)
  # Set the kcctl completion code for zsh[1] to autoload on startup
    kcctl completion zsh > "${fpath[1]}/_kcctl"`
)

var (
	completionShells = map[string]func(out io.Writer, cmd *cobra.Command) error{
		"bash": runCompletionBash,
		"zsh":  runCompletionZsh,
	}
)

func runCompletionBash(out io.Writer, kcctl *cobra.Command) error {
	return kcctl.GenBashCompletion(out)
}

func NewCmdCompletion(out io.Writer) *cobra.Command {
	var shells []string
	for s := range completionShells {
		shells = append(shells, s)
	}

	cmd := &cobra.Command{
		Use:                   "completion SHELL",
		DisableFlagsInUseLine: true,
		Short:                 "Output shell completion code for the specified shell (bash or zsh)",
		Long:                  completionLong,
		Example:               completionExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunCompletion(out, cmd, args)
			utils.CheckErr(err)
		},
		ValidArgs: shells,
	}

	return cmd
}

// RunCompletion checks given arguments and executes command
func RunCompletion(out io.Writer, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return utils.UsageErrorf(cmd, "Shell not specified.")
	}
	if len(args) > 1 {
		return utils.UsageErrorf(cmd, "Too many arguments. Expected only the shell type.")
	}
	run, found := completionShells[args[0]]
	if !found {
		return utils.UsageErrorf(cmd, "Unsupported shell type %q.", args[0])
	}

	return run(out, cmd.Parent())
}

func runCompletionZsh(out io.Writer, kcctl *cobra.Command) error {
	zshHead := "#compdef kcctl\n"

	if _, err := out.Write([]byte(zshHead)); err != nil {
		return err
	}

	zshInitialization := `
__kcctl_bash_source() {
	alias shopt=':'
	emulate -L sh
	setopt kshglob noshglob braceexpand

	source "$@"
}

__kcctl_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift

		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__kcctl_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}

__kcctl_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?

	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}

__kcctl_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}

__kcctl_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}

__kcctl_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}

__kcctl_filedir() {
	# Don't need to do anything here.
	# Otherwise we will get trailing space without "compopt -o nospace"
	true
}

autoload -U +X bashcompinit && bashcompinit

# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --version 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi

__kcctl_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__kcctl_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__kcctl_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__kcctl_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__kcctl_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__kcctl_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__kcctl_type/g" \
	<<'BASH_COMPLETION_EOF'
`
	if _, err := out.Write([]byte(zshInitialization)); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := kcctl.GenBashCompletion(buf); err != nil {
		return err
	}
	if _, err := out.Write(buf.Bytes()); err != nil {
		return err
	}

	zshTail := `
BASH_COMPLETION_EOF
}

__kcctl_bash_source <(__kcctl_convert_bash_to_zsh)
`
	_, err := out.Write([]byte(zshTail))
	return err
}
