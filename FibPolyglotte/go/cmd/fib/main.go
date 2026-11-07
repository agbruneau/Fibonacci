// Commande fib : adaptateur mince autour de la bibliothèque fib
// (contrat du PRD § 4, analyse manuelle selon l'ADR-0002).
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"fib"
)

const usage = `usage: fib [--algo iterative|matrix|fast|all] [--time] N
       fib --list [--time] N
       fib --help
`

const (
	maxTerm = 1_000_000
	maxList = 10_000
)

var comparison = []string{"fast", "iterative", "matrix"}

var errDiscordance = errors.New("discordance entre les algorithmes")

// termFunc calcule un terme : un paramètre de compute et de run, et non une table map du paquet, dont l'init au chargement ferait rougir PERF-GO-7.
type termFunc func(n int, algo string) (*big.Int, error)

type options struct {
	algos []string
	list  bool
	time  bool
	n     int
}

// parse applique la grammaire du PRD § 4.
// L'argument fautif est cité, en ASCII (%+q).
func parse(args []string) (options, error) {
	var o options
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--algo" || strings.HasPrefix(a, "--algo="):
			// --algo X : X est l'argument suivant, quel qu'il soit; chaque occurrence doit être valide,
			// la dernière gagne.
			value, ok := strings.CutPrefix(a, "--algo=")
			if !ok {
				i++
				if i == len(args) {
					return o, errors.New("--algo demande une valeur")
				}
				value = args[i]
			}
			switch {
			case value == "all":
				o.algos = comparison
			case slices.Contains(comparison, value):
				o.algos = []string{value}
			default:
				return o, fmt.Errorf("algorithme inconnu : %+q", value)
			}
		case a == "--list":
			o.list = true
		case a == "--time":
			o.time = true
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("option inconnue : %+q", a)
		default:
			positional = append(positional, a)
		}
	}
	if o.list && o.algos != nil {
		return o, errors.New("--list et --algo s'excluent")
	}
	if o.algos == nil {
		o.algos = []string{"fast"}
	}
	if len(positional) != 1 {
		return o, errors.New("exactement un indice attendu")
	}
	// Les chiffres ASCII passent avant toute conversion : Atoi accepte « +5 », et un indice trop grand pour un int
	// est hors borne.
	if !isDigits(positional[0]) {
		return o, errors.New("indice invalide : chiffres ASCII seulement")
	}
	bound := maxTerm
	if o.list {
		bound = maxList
	}
	n, err := strconv.Atoi(positional[0])
	if err != nil || n > bound {
		return o, fmt.Errorf("indice hors borne (%d au plus)", bound)
	}
	o.n = n
	return o, nil
}

// isDigits dit si s est non vide et ne compte que des chiffres ASCII (^[0-9]+$ du PRD § 4); une boucle sur les octets
// évite de compiler une regexp à chaque démarrage.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// lap rend la ligne « <algo> <durée> ms » d'un calcul : le chronomètre ne couvre que le calcul,
// et le format ignore la locale.
func lap(algo string, start time.Time) string {
	return fmt.Sprintf("%s %.3f ms\n", algo, float64(time.Since(start))/float64(time.Millisecond))
}

// compute rend les lignes à écrire (déjà en décimal pour une liste) et les lignes de durée; avec all, les trois
// algorithmes doivent concorder. Rien n'est écrit ici : une discordance (F3) laisse stdout vide et sans durée.
func compute(o options, term termFunc) ([]string, []string, error) {
	if o.list {
		start := time.Now()
		list, err := fib.FibList(o.n)
		return list, []string{lap("iterative", start)}, err
	}
	var terms []*big.Int
	var durations []string
	for _, algo := range o.algos {
		start := time.Now()
		v, err := term(o.n, algo)
		durations = append(durations, lap(algo, start))
		if err != nil {
			return nil, nil, err
		}
		terms = append(terms, v)
	}
	for _, t := range terms[1:] {
		if t.Cmp(terms[0]) != 0 {
			return nil, nil, fmt.Errorf("%w pour l'indice %d", errDiscordance, o.n)
		}
	}
	// String() et non Append dans un tampon : mesuré, les deux coûtent pareil.
	return []string{terms[0].String()}, durations, nil
}

// run rend le code de sortie : 0 succès, 2 erreur d'usage, 3 discordance.
func run(args []string, stdout, stderr io.Writer, term termFunc) int {
	// L'aide l'emporte sur tout le reste, y compris à la place de la valeur de --algo.
	if slices.Contains(args, "--help") || slices.Contains(args, "-h") {
		io.WriteString(stdout, usage)
		return 0
	}
	o, err := parse(args)
	var lines, durations []string
	if err == nil {
		lines, durations, err = compute(o, term)
	}
	if err != nil {
		fmt.Fprintf(stderr, "fib: %v\n", err)
		if errors.Is(err, errDiscordance) {
			return 3
		}
		return 2
	}
	// 64 Ko : sans tampon, --list 10000 ferait un appel système par terme; mesuré, 4 Ko coûte 12 % de plus.
	out := bufio.NewWriterSize(stdout, 64<<10)
	for _, line := range lines {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	out.Flush()
	if o.time {
		io.WriteString(stderr, strings.Join(durations, ""))
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, fib.Fib))
}
