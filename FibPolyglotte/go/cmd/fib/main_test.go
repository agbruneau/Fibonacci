// Tests de garde de la CLI (ADR-0004) : hors seams, hors catalogue. Ils visent run et ce que le binaire embarque.
package main

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"fib"
)

// writeCounter, le puits d'écriture de PERF-GO-6, 9, 10 et 12, n'implémente que Write : il garde le texte de chaque appel,
// dans l'ordre. len(chunks) est le nombre d'appels, ce qu'un appel système coûterait; size, le total des octets reçus.
type writeCounter struct{ chunks []string }

func (w *writeCounter) Write(p []byte) (int, error) {
	// string(p) copie : l'appelant, bufio.Writer en tête, réutilise p dès le retour.
	w.chunks = append(w.chunks, string(p))
	return len(p), nil
}

func (w *writeCounter) size() int {
	n := 0
	for _, chunk := range w.chunks {
		n += len(chunk)
	}
	return n
}

// minBlock : la moitié du tampon de 64 Kio de run, soit un facteur 2 de marge sur ses appels à Write.
const minBlock = 32 << 10

// PERF-GO-6, hors catalogue : garde de l'optimisation de run (écriture tamponnée) : aucun appel à Write par terme.
func TestPERFGO6_ListeEcriteParBlocs(t *testing.T) {
	var w writeCounter
	if code := run([]string{"--list", "10000"}, &w, io.Discard, fib.Fib); code != 0 {
		t.Fatalf("run --list 10000 rend le code %d, attendu 0", code)
	}
	size := w.size()
	if size == 0 {
		t.Fatal("run --list 10000 n'écrit aucun octet sur stdout")
	}
	maxCalls := (size + minBlock - 1) / minBlock
	t.Run(fmt.Sprintf("PERF-GO-6 : --list 10000 écrit sur stdout par blocs d'au moins %d octets", minBlock), func(t *testing.T) {
		if calls := len(w.chunks); calls > maxCalls {
			t.Errorf("--list 10000 fait %d appels à Write pour %d octets, attendu %d au plus", calls, size, maxCalls)
		}
	})
}

// PERF-GO-7, hors catalogue : garde de l'optimisation de démarrage : aucune initialisation dans fib ni dans la commande
// (le sync.Once de quads, aucune regexp compilée au chargement). Le binaire de test se relance sans test, sous
// GODEBUG=inittrace=1 : le runtime écrit sur stderr une ligne « init <paquet> » pour chaque paquet qui initialise
// quelque chose. La ligne « init main », celle du paquet de test généré, n'est pas la commande.
func TestPERFGO7_LeDemarrageNInitialiseRien(t *testing.T) {
	// L'instrumentation de couverture ajoute un init à fib/cmd/fib (et à fib sous -coverpkg) : la garde les prendrait pour les nôtres.
	if testing.CoverMode() != "" {
		t.Skip("PERF-GO-7 saute sous couverture : l'instrumentation ajoute un init à fib/cmd/fib (et à fib sous -coverpkg)")
	}
	t.Run("PERF-GO-7 : le chargement de fib et de la commande n'exécute aucune initialisation", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=^$")
		cmd.Env = append(os.Environ(), "GODEBUG=inittrace=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		traced := false
		for line := range strings.Lines(string(out)) {
			traced = traced || strings.HasPrefix(line, "init ")
			if strings.HasPrefix(line, "init fib ") || strings.HasPrefix(line, "init fib/cmd/fib ") {
				t.Errorf("initialisation au chargement : %s", strings.TrimSpace(line))
			}
		}
		if !traced {
			t.Fatalf("aucune ligne « init » : GODEBUG=inittrace=1 est sans effet\n%s", out)
		}
	})
}

// PERF-GO-8, hors catalogue : garde de l'optimisation de démarrage : aucune regexp, pendant de PERF-PY-8. Un paquet
// importé est dans les dépendances de la commande, qu'il serve à l'initialisation (PERF-GO-7 le voit) ou à chaque
// appel de isDigits (PERF-GO-7 ne le voit pas).
func TestPERFGO8_LaCommandeNImporteAucuneRegexp(t *testing.T) {
	t.Run("PERF-GO-8 : go list -deps ne contient aucun paquet regexp", func(t *testing.T) {
		out, err := exec.Command("go", "list", "-deps", ".").Output()
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				err = fmt.Errorf("%w\n%s", err, exit.Stderr)
			}
			t.Fatalf("go list -deps . : %v", err)
		}
		deps := strings.Fields(string(out))
		if !slices.Contains(deps, "fib") {
			t.Fatalf("go list -deps . ne liste pas fib : %q", deps)
		}
		for _, dep := range deps {
			if dep == "regexp" || strings.HasPrefix(dep, "regexp/") {
				t.Errorf("la commande dépend de %s", dep)
			}
		}
	})
}

// PERF-GO-9, hors catalogue : garde de F3 (catalogue § 6). Aucun cas du contrat ne peut provoquer une discordance : seul un
// calcul substitué, le vrai terme plus 1 pour un algorithme, tour à tour à chacune des trois places, la provoque. run rend
// alors 3, stdout ne reçoit aucun appel à Write, et stderr un seul : une ligne « fib: », sans durée, même avec --time. Le
// test ne recalcule aucun terme. Rouge sous : un prédicat aveugle (t.Cmp(t)) ou partiel (terms[1:2]), `return 2` pour le 3,
// des durées ajoutées au message (une par ligne, ou sur sa ligne), le terme écrit sur le stdout de run avant l'erreur, ou
// fib.Fib en dur dans compute.
// Survivants connus, laissés à la revue de code : une écriture directe sur os.Stderr ou os.Stdout depuis compute (G9 et
// G10, que le Writer d'essai ne voit pas); main qui borne le code à 2 (GM); main qui passe un calcul qui fait tout par
// fast (GW).
func TestPERFGO9_UneDiscordanceRend3SansStdoutNiDuree(t *testing.T) {
	for _, bad := range comparison {
		t.Run(fmt.Sprintf("PERF-GO-9 : si %s diffère des deux autres, --time --algo all 10 rend 3, sans stdout et sans durée", bad), func(t *testing.T) {
			term := func(n int, algo string) (*big.Int, error) {
				v, err := fib.Fib(n, algo)
				if err != nil || algo != bad {
					return v, err
				}
				return new(big.Int).Add(v, big.NewInt(1)), nil
			}
			var stdout, stderr writeCounter
			if code := run([]string{"--time", "--algo", "all", "10"}, &stdout, &stderr, term); code != 3 {
				t.Errorf("run --time --algo all 10 rend le code %d, attendu 3", code)
			}
			if len(stdout.chunks) != 0 {
				t.Errorf("stdout reçoit %d appels à Write, attendu aucun : %q", len(stdout.chunks), stdout.chunks)
			}
			if len(stderr.chunks) != 1 {
				t.Fatalf("stderr reçoit %d appels à Write, attendu un seul : %q", len(stderr.chunks), stderr.chunks)
			}
			line := stderr.chunks[0]
			if !strings.HasPrefix(line, "fib: ") {
				t.Errorf("stderr ne commence pas par « fib: » : %q", line)
			}
			if strings.Count(line, "\n") != 1 || !strings.HasSuffix(line, "\n") {
				t.Errorf("stderr n'est pas une seule ligne, terminée par son seul \\n : %q", line)
			}
			if strings.Contains(line, " ms") {
				t.Errorf("stderr contient une durée : %q", line)
			}
		})
	}
}

// PERF-GO-10, hors catalogue : garde de F2 et de F6 (catalogue § 6), pendant de PERF-TS-8 et de PERF-TS-12 : l'algorithme que
// nomme une ligne de durée est celui qui a calculé. Le calcul espion note l'algorithme qu'on lui demande et rend
// big.NewInt(0), sans calculer : les algorithmes appelés doivent être ceux que nomment les lignes de durée, dans l'ordre, un
// appel chacun. Chaque cas est joué à n = 10 et à n = maxTerm (1000000, la borne F8) : sans calcul, la borne ne coûte rien, et
// un nom passé au calcul qui dépend de n se voit aux deux bouts de la plage. stdout reçoit le seul `0\n`, le terme que
// l'espion a rendu. Rouge sous : un algorithme calculé par un autre (fast pour iterative, par exemple) que la ligne nomme
// quand même, et, dans compute, `name := algo; if o.n > 10 && name == "matrix" { name = "fast" }` puis `term(o.n, name)`, la
// ligne de durée gardant algo (vert avant que la garde joue n = maxTerm).
// Survivant connu, laissé à la revue de code : un aiguillage qui dépend de n hors de ces deux indices, tel que
// `o.n == 12345` ou `o.n > 10 && o.n < maxTerm`.
func TestPERFGO10_LAlgorithmeNommeEstCeluiQuiCalcule(t *testing.T) {
	for _, n := range []int{10, maxTerm} {
		for _, c := range []struct {
			options []string
			algos   []string
		}{
			{nil, []string{"fast"}},
			{[]string{"--algo", "iterative"}, []string{"iterative"}},
			{[]string{"--algo", "matrix"}, []string{"matrix"}},
			{[]string{"--algo", "all"}, []string{"fast", "iterative", "matrix"}},
		} {
			args := append(append([]string{"--time"}, c.options...), strconv.Itoa(n))
			t.Run(fmt.Sprintf("PERF-GO-10 : %s exécute %s", strings.Join(args, " "), strings.Join(c.algos, " puis ")), func(t *testing.T) {
				var called []string
				term := func(_ int, algo string) (*big.Int, error) {
					called = append(called, algo)
					return big.NewInt(0), nil
				}
				var stdout, stderr writeCounter
				if code := run(args, &stdout, &stderr, term); code != 0 {
					t.Fatalf("run %s rend le code %d, attendu 0", strings.Join(args, " "), code)
				}
				if got := strings.Join(stdout.chunks, ""); got != "0\n" {
					t.Errorf("stdout reçoit %q, attendu le terme de l'espion, %q", got, "0\n")
				}
				var named []string
				for _, chunk := range stderr.chunks {
					for line := range strings.Lines(chunk) {
						algo, _, _ := strings.Cut(line, " ")
						named = append(named, algo)
					}
				}
				if !slices.Equal(named, c.algos) {
					t.Errorf("les lignes de durée nomment %q, attendu %q", named, c.algos)
				}
				if !slices.Equal(called, named) {
					t.Errorf("les algorithmes appelés sont %q, les lignes de durée nomment %q", called, named)
				}
			})
		}
	}
}

// sleepSpan : la durée que dort le calcul lent de PERF-GO-12. roundHalf : la demi-unité de %.3f, en ms, l'erreur d'arrondi
// d'une ligne de durée. floatSlack : la marge de l'arithmétique flottante, en ms, pour une égalité exacte à l'arrondi près.
const (
	sleepSpan  = 20 * time.Millisecond
	roundHalf  = 0.0005
	floatSlack = 1e-9
)

// PERF-GO-12, hors catalogue : garde de F6 (catalogue § 6, lacune « la durée est mesurée »), pendant de PERF-RS-14 et de
// PERF-PY-31. Le contrat ne lit que le format des lignes de durée : `0.000 ms` y passe. La garde joue trois fois
// `--time --algo all 10`; à chaque tour, un seul algorithme est lent, tour à tour fast, iterative et matrix. Un calcul
// espion, injecté comme celui de PERF-GO-10 (le termFunc que reçoit run), dort alors 20 ms pour lui seul, puis délègue à
// fib.Fib, et chronomètre son propre sommeil par l'horloge même de la production (time.Now et time.Since lisent, comme
// time.Sleep, la même horloge monotone).
//
// Trois assertions. (1) Le témoin : le sommeil chronométré vaut au moins 10 ms, sans quoi la garde serait à vide.
// (2) La ligne de l'algorithme lent, plus 0,0005 ms (l'arrondi de %.3f), vaut au moins le sommeil chronométré : les quatre
// lectures d'horloge (début de la fenêtre de production, début et fin du sommeil de l'espion, fin de la fenêtre) sont
// emboîtées sur un même fil et une même horloge monotone, donc l'inégalité est exacte, sans dépendre de la précision du
// sommeil ni de la charge, y compris sous Windows. (3) La somme des trois lignes vaut au plus le temps mural de l'appel de
// run, lu par la même horloge, plus 3 × 0,0005 ms : les trois fenêtres sont disjointes et comprises dans l'appel.
//
// Rouge sous : une durée constante (0.000, ou une constante réaliste comme 1.234); le chronomètre lancé après le calcul;
// des secondes ou des microsecondes au lieu de millisecondes; un départ commun à tous les algorithmes (durées cumulées);
// des durées permutées entre algorithmes, les noms justes et les valeurs décalées d'un cran.
// Survivants connus, laissés à la revue de code : l'étendue exacte de la fenêtre (une fenêtre qui engloberait aussi la mise
// en forme du terme passerait); la fenêtre de --list, que FibList, appelé sans calcul en paramètre, ne laisse pas ralentir
// (une borne du vingtième du temps mural y serait instable sous Windows, où nanotime avance par ticks, jusqu'à 15,6 ms,
// alors que FibList(10000) prend environ 4 ms).
func TestPERFGO12_LaDureeEstMesuree(t *testing.T) {
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	for _, slow := range comparison {
		t.Run(fmt.Sprintf("PERF-GO-12 : si %s dort %d ms, sa ligne de durée le couvre et les trois lignes tiennent dans l'appel", slow, sleepSpan.Milliseconds()), func(t *testing.T) {
			var slept time.Duration
			term := func(n int, algo string) (*big.Int, error) {
				if algo == slow {
					begin := time.Now()
					time.Sleep(sleepSpan)
					slept = time.Since(begin)
				}
				return fib.Fib(n, algo)
			}
			var stderr writeCounter
			begin := time.Now()
			code := run([]string{"--time", "--algo", "all", "10"}, io.Discard, &stderr, term)
			wall := time.Since(begin)
			if code != 0 {
				t.Fatalf("run --time --algo all 10 rend le code %d, attendu 0", code)
			}
			if slept < sleepSpan/2 {
				t.Fatalf("le calcul lent de %s n'a dormi que %v pour %v demandés : la garde serait à vide", slow, slept, sleepSpan)
			}
			var names []string
			var slowMs, total float64
			for _, chunk := range stderr.chunks {
				for line := range strings.Lines(chunk) {
					fields := strings.Fields(line)
					if len(fields) != 3 || fields[2] != "ms" {
						t.Fatalf("ligne de durée illisible : %q", line)
					}
					d, err := strconv.ParseFloat(fields[1], 64)
					if err != nil {
						t.Fatalf("ligne de durée illisible : %q (%v)", line, err)
					}
					names = append(names, fields[0])
					total += d
					if fields[0] == slow {
						slowMs = d
					}
				}
			}
			if !slices.Equal(names, comparison) {
				t.Fatalf("les lignes de durée nomment %q, attendu %q", names, comparison)
			}
			if slowMs+roundHalf+floatSlack < ms(slept) {
				t.Errorf("la ligne de %s donne %.3f ms, moins que les %.3f ms que le calcul a dormi dans sa fenêtre", slow, slowMs, ms(slept))
			}
			if limit := ms(wall) + 3*roundHalf + floatSlack; total > limit {
				t.Errorf("les trois lignes totalisent %.3f ms, plus que les %.3f ms de l'appel de run : les fenêtres se recouvrent ou l'unité est fausse", total, ms(wall))
			}
		})
	}
}
