// Package cli provides shell completion script generation for various shells.
package cli

import (
	"fmt"
	"io"
)

// GenerateCompletion generates a shell completion script for the specified shell.
//
// Parameters:
//   - out: The writer to output the completion script.
//   - shell: The shell type ("bash", "zsh", "fish", "powershell").
//   - algorithms: List of available algorithm names.
//
// Returns:
//   - error: An error if the shell is not supported.
func GenerateCompletion(out io.Writer, shell string, algorithms []string) error {
	switch shell {
	case "bash":
		return generateBashCompletion(out, algorithms)
	case "zsh":
		return generateZshCompletion(out, algorithms)
	case "fish":
		return generateFishCompletion(out, algorithms)
	case "powershell", "ps":
		return generatePowerShellCompletion(out, algorithms)
	default:
		return fmt.Errorf("unsupported shell: %s (accepted values: bash, zsh, fish, powershell)", shell)
	}
}

// generateBashCompletion generates a Bash completion script.
func generateBashCompletion(out io.Writer, algorithms []string) error {
	script := `# Bash completion script for fibcalc
# Add this to your ~/.bashrc or ~/.bash_completion

_fibcalc_completions() {
    local cur prev opts algorithms
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main options
    opts="--help -h --version -V -n -v -d --details --timeout --algo --threshold --fft-threshold --strassen-threshold --calibrate --auto-calibrate --calibration-profile --json --server --port --no-color --output -o --quiet -q --hex --interactive --completion"

    # Available algorithms
    algorithms="%s all"

    case "${prev}" in
        --algo)
            COMPREPLY=( $(compgen -W "${algorithms}" -- "${cur}") )
            return 0
            ;;
        --completion)
            COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- "${cur}") )
            return 0
            ;;
        --output|-o|--calibration-profile)
            # File/directory completion
            COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
        --port)
            COMPREPLY=( $(compgen -W "8080 3000 5000 9000" -- "${cur}") )
            return 0
            ;;
        --timeout)
            COMPREPLY=( $(compgen -W "1m 5m 10m 30m 1h" -- "${cur}") )
            return 0
            ;;
        --threshold|--fft-threshold|--strassen-threshold)
            COMPREPLY=( $(compgen -W "1024 2048 4096 8192 16384" -- "${cur}") )
            return 0
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi
}

complete -F _fibcalc_completions fibcalc
`
	algoList := ""
	for i, algo := range algorithms {
		if i > 0 {
			algoList += " "
		}
		algoList += algo
	}

	_, err := fmt.Fprintf(out, script, algoList)
	return err
}

// generateZshCompletion generates a Zsh completion script.
func generateZshCompletion(out io.Writer, algorithms []string) error {
	script := `#compdef fibcalc

# Zsh completion script for fibcalc
# Add this to your ~/.zshrc or place in $fpath

_fibcalc() {
    local -a algorithms
    algorithms=(%s all)

    _arguments -s \
        '(-h --help)'{-h,--help}'[Show help message]' \
        '(-V --version)'{-V,--version}'[Show version information]' \
        '-n[Index n of Fibonacci number]:number:' \
        '-v[Display full result value]' \
        '(-d --details)'{-d,--details}'[Show performance details]' \
        '--timeout[Maximum execution time]:duration:(1m 5m 10m 30m 1h)' \
        '--algo[Algorithm to use]:algorithm:($algorithms)' \
        '--threshold[Parallelism threshold in bits]:bits:(1024 2048 4096 8192 16384)' \
        '--fft-threshold[FFT threshold in bits]:bits:(100000 500000 1000000)' \
        '--strassen-threshold[Strassen threshold in bits]:bits:(1024 2048 3072 4096)' \
        '--calibrate[Run calibration mode]' \
        '--auto-calibrate[Enable auto-calibration]' \
        '--calibration-profile[Calibration profile file]:file:_files' \
        '--json[Output in JSON format]' \
        '--server[Start HTTP server mode]' \
        '--port[Server port]:port:(8080 3000 5000 9000)' \
        '--no-color[Disable colored output]' \
        '(-o --output)'{-o,--output}'[Output file path]:file:_files' \
        '(-q --quiet)'{-q,--quiet}'[Quiet mode for scripts]' \
        '--hex[Display result in hexadecimal]' \
        '--interactive[Start interactive REPL mode]' \
        '--completion[Generate completion script]:shell:(bash zsh fish powershell)'
}

_fibcalc "$@"
`
	algoList := ""
	for i, algo := range algorithms {
		if i > 0 {
			algoList += " "
		}
		algoList += algo
	}

	_, err := fmt.Fprintf(out, script, algoList)
	return err
}

// generateFishCompletion generates a Fish completion script.
func generateFishCompletion(out io.Writer, algorithms []string) error {
	script := `# Fish completion script for fibcalc
# Add this to ~/.config/fish/completions/fibcalc.fish

# Disable file completion by default
complete -c fibcalc -f

# Help and version
complete -c fibcalc -s h -l help -d 'Show help message'
complete -c fibcalc -s V -l version -d 'Show version information'

# Main options
complete -c fibcalc -s n -d 'Fibonacci index to calculate' -x
complete -c fibcalc -s v -d 'Display full result value'
complete -c fibcalc -s d -l details -d 'Show performance details'
complete -c fibcalc -l timeout -d 'Maximum execution time' -xa '1m 5m 10m 30m 1h'
complete -c fibcalc -l algo -d 'Algorithm to use' -xa '%s all'
complete -c fibcalc -l threshold -d 'Parallelism threshold in bits' -xa '1024 2048 4096 8192 16384'
complete -c fibcalc -l fft-threshold -d 'FFT threshold in bits' -xa '100000 500000 1000000'
complete -c fibcalc -l strassen-threshold -d 'Strassen threshold' -xa '1024 2048 3072 4096'

# Calibration
complete -c fibcalc -l calibrate -d 'Run calibration mode'
complete -c fibcalc -l auto-calibrate -d 'Enable auto-calibration'
complete -c fibcalc -l calibration-profile -d 'Calibration profile file' -rF

# Output options
complete -c fibcalc -l json -d 'Output in JSON format'
complete -c fibcalc -s o -l output -d 'Output file path' -rF
complete -c fibcalc -s q -l quiet -d 'Quiet mode for scripts'
complete -c fibcalc -l hex -d 'Display result in hexadecimal'
complete -c fibcalc -l no-color -d 'Disable colored output'

# Server mode
complete -c fibcalc -l server -d 'Start HTTP server mode'
complete -c fibcalc -l port -d 'Server port' -xa '8080 3000 5000 9000'

# Interactive and completion
complete -c fibcalc -l interactive -d 'Start interactive REPL mode'
complete -c fibcalc -l completion -d 'Generate completion script' -xa 'bash zsh fish powershell'
`
	algoList := ""
	for i, algo := range algorithms {
		if i > 0 {
			algoList += " "
		}
		algoList += algo
	}

	_, err := fmt.Fprintf(out, script, algoList)
	return err
}

// generatePowerShellCompletion generates a PowerShell completion script.
func generatePowerShellCompletion(out io.Writer, algorithms []string) error {
	script := `# PowerShell completion script for fibcalc
# Add this to your $PROFILE

$fibcalcAlgorithms = @(%s, 'all')

Register-ArgumentCompleter -CommandName 'fibcalc' -Native -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $options = @(
        @{Name = '-h'; Description = 'Show help message' }
        @{Name = '--help'; Description = 'Show help message' }
        @{Name = '-V'; Description = 'Show version information' }
        @{Name = '--version'; Description = 'Show version information' }
        @{Name = '-n'; Description = 'Fibonacci index to calculate' }
        @{Name = '-v'; Description = 'Display full result value' }
        @{Name = '-d'; Description = 'Show performance details' }
        @{Name = '--details'; Description = 'Show performance details' }
        @{Name = '--timeout'; Description = 'Maximum execution time' }
        @{Name = '--algo'; Description = 'Algorithm to use' }
        @{Name = '--threshold'; Description = 'Parallelism threshold in bits' }
        @{Name = '--fft-threshold'; Description = 'FFT threshold in bits' }
        @{Name = '--strassen-threshold'; Description = 'Strassen threshold' }
        @{Name = '--calibrate'; Description = 'Run calibration mode' }
        @{Name = '--auto-calibrate'; Description = 'Enable auto-calibration' }
        @{Name = '--calibration-profile'; Description = 'Calibration profile file' }
        @{Name = '--json'; Description = 'Output in JSON format' }
        @{Name = '--server'; Description = 'Start HTTP server mode' }
        @{Name = '--port'; Description = 'Server port' }
        @{Name = '--no-color'; Description = 'Disable colored output' }
        @{Name = '-o'; Description = 'Output file path' }
        @{Name = '--output'; Description = 'Output file path' }
        @{Name = '-q'; Description = 'Quiet mode for scripts' }
        @{Name = '--quiet'; Description = 'Quiet mode for scripts' }
        @{Name = '--hex'; Description = 'Display result in hexadecimal' }
        @{Name = '--interactive'; Description = 'Start interactive REPL mode' }
        @{Name = '--completion'; Description = 'Generate completion script' }
    )

    $elements = $commandAst.CommandElements
    $lastElement = if ($elements.Count -gt 1) { $elements[-1].ToString() } else { '' }
    $prevElement = if ($elements.Count -gt 2) { $elements[-2].ToString() } else { '' }

    # Context-aware completions
    switch ($prevElement) {
        '--algo' {
            $fibcalcAlgorithms | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }
        '--completion' {
            @('bash', 'zsh', 'fish', 'powershell') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }
        '--timeout' {
            @('1m', '5m', '10m', '30m', '1h') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }
        '--port' {
            @('8080', '3000', '5000', '9000') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }
    }

    # Default: show options
    $options | Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_.Name, $_.Name, 'ParameterName', $_.Description)
    }
}
`
	algoList := ""
	for i, algo := range algorithms {
		if i > 0 {
			algoList += ", "
		}
		algoList += fmt.Sprintf("'%s'", algo)
	}

	_, err := fmt.Fprintf(out, script, algoList)
	return err
}
