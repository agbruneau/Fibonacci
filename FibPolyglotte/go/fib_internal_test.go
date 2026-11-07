// Tests de garde qui visent une fonction non exportée (ADR-0004) : hors seams, hors catalogue.
package fib

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// PERF-GO-5, hors catalogue : garde de l'optimisation de FibList (additionneur en base 10^18), pendant de PERF-RS-4.
func TestPERFGO5_UneSommeDeLimbesEgaleA10Puissance18PropageSaRetenue(t *testing.T) {
	t.Run("PERF-GO-5 : [base/2, 3, 0] + [base/2, 4] vaut [0, 8, 0], sans retenue finale", func(t *testing.T) {
		a := []uint64{base / 2, 3, 0}
		carry := addLimbs(a, []uint64{base / 2, 4})
		if want := []uint64{0, 8, 0}; !slices.Equal(a, want) || carry != 0 {
			t.Errorf("[base/2, 3, 0] + [base/2, 4] = %v avec la retenue %d, attendu %v avec la retenue 0", a, carry, want)
		}
	})
}

// identityRuns : le nombre d'exécutions que moyenne testing.AllocsPerRun. Le résultat est une division entière, donc une
// allocation parasite isolée (le ramasse-miettes, le démarrage d'une goroutine) y disparaît, comme sous PERF-GO-3 et 4.
const identityRuns = 20

// identitySink reçoit chaque résultat que mesure PERF-GO-11 : sans lui, un résultat inutilisé pourrait être alloué sur la
// pile, ou le calcul éliminé, et le compte ne serait plus celui de la fonction.
var identitySink *big.Int

// PERF-GO-11, hors catalogue : garde de F2 et de F6 (catalogue § 6, lacune « l'aiguillage de la bibliothèque »), pendant de
// PERF-GO-10, qui espionne le nom demandé et non la fonction qui calcule derrière ce nom. Les trois algorithmes rendent les
// mêmes valeurs : sous `case "matrix": return fast(n), nil`, les valeurs restent justes. La garde ne compare donc ni valeur
// ni durée : elle compare le nombre exact d'allocations de Fib(n, nom) à celui de l'appel direct de la fonction de ce nom,
// car chaque algorithme alloue à sa façon (iterative réserve ses tampons d'emblée; matrix et fast créent des entiers à
// chaque bit, en nombres différents). Elle rougit sous chacun des six échanges d'aiguillage du `switch algo` de Fib, un nom
// vers l'une des deux autres fonctions : iterative → matrix, iterative → fast, matrix → iterative, matrix → fast,
// fast → iterative et fast → matrix. Le code de production ne change pas : le `switch` reste tel quel, et la garde passe
// par Fib elle-même, pas par un détour qui ne servirait qu'à elle.
//
// n = 1023 : tous ses bits valent 1, donc matrix ajoute à chaque bit son produit par Q, que fast n'a pas; les trois comptes
// valent 4, 67 et 51 sous Go 1.26. La garde vérifie d'abord qu'ils diffèrent deux à deux : si une évolution de math/big ou
// d'un algorithme les confondait, elle échouerait sur ce constat au lieu de ne plus rien garder.
//
// Limite : l'empreinte ne vérifie l'aiguillage qu'à n = 1023, et un corps qui en imite une autre la trompe : un `matrix` qui
// calcule par fast puis alloue 16 entiers de plus retrouve les 67 allocations de matrix. Le volet statique ci-dessous lit
// donc le texte de fib.go, et ferme ce que n = 1023 laisse passer tant que le texte nomme la fonction vers laquelle il
// aiguille et que algo et n, qui choisissent le cas et l'entier, restent les paramètres de Fib. Ce qui reste, rejoué et laissé
// à la revue de code (la liste dit ce qu'on a rejoué, non que rien d'autre ne passe) : un calcul que le texte ne rattache à
// aucun nom d'algorithme, par une variable de fonction (`var delegate = fast`, puis `return delegate(n)`), par une fonction
// auxiliaire (`return help(n)`, où help appelle fast) ou par la copie en ligne du code d'un algorithme (celui de fast, dans
// `if n > 10_000 { … }`), le tout dans le corps d'un autre algorithme, ou dans Fib hors du `switch algo`. Il survit à tout n
// quand ce corps imite aussi l'empreinte d'allocations (67 pour matrix, comme le `matrix` ci-dessus : le calcul de fast, 51,
// plus 16 entiers), et, sans cette imitation, pour certains n seulement : tout n autre que 1023, par exemple
// `if n > 10_000 { return help(n) }`.
func TestPERFGO11_ChaqueNomAiguilleVersSaFonction(t *testing.T) {
	const n = 1023
	algos := []struct {
		name   string
		direct func(int) *big.Int
	}{{"iterative", iterative}, {"matrix", matrix}, {"fast", fast}}
	own := make([]float64, len(algos)) // le compte de chaque fonction, appelée directement
	for i, a := range algos {
		own[i] = testing.AllocsPerRun(identityRuns, func() { identitySink = a.direct(n) })
	}
	for i := range algos {
		for j := i + 1; j < len(algos); j++ {
			if own[i] == own[j] {
				t.Fatalf("%s(%d) et %s(%d) allouent tous deux %v fois : l'un délègue-t-il à l'autre (identité perdue, ADR-0004) ? "+
					"Sinon, choisir un autre n ou une autre empreinte", algos[i].name, n, algos[j].name, n, own[i])
			}
		}
	}
	for i, a := range algos {
		t.Run(fmt.Sprintf("PERF-GO-11 : Fib(%d, %q) exécute %s", n, a.name, a.name), func(t *testing.T) {
			got := testing.AllocsPerRun(identityRuns, func() {
				var err error
				if identitySink, err = Fib(n, a.name); err != nil {
					t.Fatalf("Fib(%d, %q) : %v", n, a.name, err)
				}
			})
			if got == own[i] {
				return
			}
			who := "le compte d'aucune des trois fonctions"
			for j, b := range algos {
				if got == own[j] {
					who = "le compte de " + b.name
				}
			}
			t.Errorf("Fib(%d, %q) alloue %v fois, soit %s; %s en alloue %v", n, a.name, got, who, a.name, own[i])
		})
	}
}

// PERF-GO-11, volet statique, hors catalogue : garde de F2, qui veut que chaque nom calcule par son propre algorithme, à
// tout n. L'empreinte d'allocations ne le voit qu'à n = 1023, et un corps qui l'imite passe. Ce volet lit fib.go par
// go/parser et go/ast (la bibliothèque standard) et vérifie trois choses. (a) Dans Fib, le `switch algo` a les trois cas
// "iterative", "matrix" et "fast", et le corps de chacun est la seule instruction `return <f>(n), nil`, où <f> est la
// fonction du même nom; une clause `default` est admise si son corps ne nomme aucun des trois algorithmes (le retour d'erreur
// final de Fib peut y être déplacé). (b) Les corps de iterative, matrix et fast ne nomment (aucun *ast.Ident) ni l'une des
// deux autres fonctions, ni Fib, ni FibList. (c) Dans tout le corps de Fib, switch compris ou non, chacun des noms
// iterative, matrix et fast n'apparaît qu'une fois, dans le retour de son propre cas; Fib et FibList n'y sont jamais
// nommés; chaque occurrence de algo ou de n désigne le paramètre de Fib du même nom, dont le parser résout l'objet (`var algo`,
// `algo :=` ou `for _, algo := range` dans un bloc qui entoure le switch en déclarent un autre, qui masque le paramètre); et
// aucune de ces occurrences n'est la cible d'une écriture (partie gauche d'une affectation, `++` ou `--`, clé ou valeur d'un
// `range`, entre parenthèses ou non) ni l'opérande d'un `&`, par où un pointeur réaffecterait le paramètre. Un aiguillage posé
// hors du switch, avant lui par exemple, rougit donc dès qu'il nomme un algorithme de plus, rappelle Fib, masque, réaffecte ou
// adresse algo ou n. Le code de production ne change pas, et les imports de ce fichier n'entrent ni dans `go list -deps .` ni
// dans l'inittrace de cmd/fib (PERF-GO-7 et 8 y restent verts).
//
// Rouge sous : `case "matrix": if n > 10_000 { return fast(n), nil }; return matrix(n), nil` (a), ou la même condition
// `if n%2 == 0` (a); `return fast(n)` en tête de matrix (b, et l'empreinte); un matrix qui calcule par fast puis alloue
// 16 entiers (une variable de paquet que `new(int)` remplit) pour retrouver les 67 allocations de matrix (b); et
// `if n < 64 { return iterative(n) }` en tête de fast (b), rouge par intention : même juste et plus rapide, il fait calculer
// fast par iterative, et F2 veut un calcul par son propre algorithme à tout n. Puis, avant le switch de Fib, sous la
// condition `algo == "matrix" && n > 10_000` : `{ return fast(n), nil }` (c, fast nommé deux fois), `{ algo = "fast" }` (c,
// algo réaffecté), `{ (algo) = "fast" }` (c, cible entre parenthèses), `{ for _, algo = range []string{"fast"} {} }` (c, algo
// réaffecté par un range), `{ return Fib(n, "fast") }` (c, Fib nommé) et, après `p := &algo`, `{ *p = "fast" }` (c, adresse
// prise). Puis, avec `alt := map[bool]string{true: "fast", false: algo}[algo == "matrix" && n > 10_000]`, le switch tout
// entier dans `{ var algo = alt; switch algo { … } }` ou dans `for _, algo := range []string{alt} { switch algo { … } }`
// (c, algo masqué).
// Reste vert, par intention : le `return nil, fmt.Errorf(...)` final de Fib déplacé dans une clause `default`.
// Survivants connus, laissés à la revue de code : voir la limite de PERF-GO-11 ci-dessus.
func TestPERFGO11_LAiguillageEtLesCorpsSeLisentDansLeTexte(t *testing.T) {
	algos := []string{"iterative", "matrix", "fast"}
	fset, funcs := fibFuncs(t)
	for _, name := range append([]string{"Fib", "FibList"}, algos...) {
		if funcs[name] == nil {
			t.Fatalf("fib.go ne déclare pas la fonction %s : renommée ? Alors adapter PERF-GO-11", name)
		}
	}

	t.Run("PERF-GO-11 : le switch algo de Fib a les cas iterative, matrix et fast, chacun réduit à return <nom>(n), nil", func(t *testing.T) {
		var switches []*ast.SwitchStmt
		ast.Inspect(funcs["Fib"].Body, func(node ast.Node) bool {
			if sw, ok := node.(*ast.SwitchStmt); ok && isIdent(sw.Tag, "algo") {
				switches = append(switches, sw)
			}
			return true
		})
		if len(switches) != 1 {
			t.Fatalf("Fib compte %d switch algo, attendu un seul", len(switches))
		}
		var cases []string
		for _, stmt := range switches[0].Body.List {
			clause := stmt.(*ast.CaseClause) // le corps d'un switch ne compte que des clauses
			where := fset.Position(clause.Pos())
			if clause.List == nil { // default : seul un nom inconnu l'atteint, donc elle ne doit aiguiller vers aucun des trois
				for _, body := range clause.Body {
					ast.Inspect(body, func(node ast.Node) bool {
						if id, ok := node.(*ast.Ident); ok && slices.Contains(algos, id.Name) {
							t.Errorf("%s : la clause default nomme %s", fset.Position(id.Pos()), id.Name)
						}
						return true
					})
				}
				continue
			}
			lit, ok := clauseLiteral(clause)
			if !ok {
				t.Errorf("%s : une clause n'est pas un seul nom d'algorithme entre guillemets (plusieurs valeurs, ou autre chose qu'une chaîne)", where)
				continue
			}
			name, _ := strconv.Unquote(lit.Value)
			cases = append(cases, name)
			if !returnsOwnCall(clause.Body, name) {
				t.Errorf("%s : le cas %q n'est pas la seule instruction `return %s(n), nil`", where, name, name)
			}
		}
		slices.Sort(cases)
		if want := []string{"fast", "iterative", "matrix"}; !slices.Equal(cases, want) {
			t.Errorf("le switch algo de Fib a les cas %q, attendu exactement %q", cases, want)
		}
	})

	t.Run("PERF-GO-11 : le corps de Fib nomme chaque algorithme une fois, ni Fib ni FibList, et ne masque, n'affecte ni n'adresse algo ou n", func(t *testing.T) {
		watched := append([]string{"Fib", "FibList"}, algos...)
		// Ident.Obj est déprécié au profit de go/types, mais ParseFile le remplit en mode 0 : le parser rattache chaque
		// occurrence de algo ou de n à sa déclaration. Un paramètre sans objet (mode SkipObjectResolution) échoue ci-dessous
		// au lieu de laisser deux Obj nil se ressembler.
		params := map[string]*ast.Object{} // algo et n : l'objet du paramètre de Fib du même nom
		for _, field := range funcs["Fib"].Type.Params.List {
			for _, id := range field.Names {
				if id.Name == "algo" || id.Name == "n" {
					params[id.Name] = id.Obj
				}
			}
		}
		for _, name := range []string{"algo", "n"} {
			if params[name] == nil {
				t.Fatalf("Fib n'a pas de paramètre %s avec un objet résolu : renommé ? Alors adapter PERF-GO-11", name)
			}
		}
		// flag signale chaque expression qui est algo ou n, parenthèses défaites : `(algo) = x` écrit algo comme `algo = x`.
		// Pas de déréférencement à défaire : `*algo` et `*n` ne compilent pas (string et int), et `*p = x` écrit la valeur
		// pointée, non p. Pour `p := &algo; *p = "fast"`, c'est le contrôle de l'adresse prise (`&algo`) qui rougit.
		flag := func(verb string, exprs ...ast.Expr) {
			for _, e := range exprs {
				if id, ok := ast.Unparen(e).(*ast.Ident); ok && params[id.Name] != nil {
					t.Errorf("%s : Fib %s %s", fset.Position(id.Pos()), verb, id.Name)
				}
			}
		}
		named := map[string][]string{} // pour chaque nom surveillé, la position de chacune de ses occurrences
		ast.Inspect(funcs["Fib"].Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.Ident:
				if slices.Contains(watched, node.Name) {
					named[node.Name] = append(named[node.Name], fset.Position(node.Pos()).String())
				}
				if obj, ok := params[node.Name]; ok && node.Obj != obj {
					t.Errorf("%s : %s n'est pas le paramètre de Fib (redéclaré, masqué, ou autre chose qui porte ce nom)", fset.Position(node.Pos()), node.Name)
				}
			case *ast.AssignStmt:
				flag("affecte", node.Lhs...)
			case *ast.IncDecStmt:
				flag("affecte", node.X)
			case *ast.RangeStmt:
				flag("affecte", node.Key, node.Value) // l'un ou l'autre est nil quand le range ne les écrit pas
			case *ast.UnaryExpr:
				if node.Op == token.AND {
					flag("prend l'adresse de", node.X)
				}
			}
			return true
		})
		for _, name := range []string{"Fib", "FibList"} {
			if where := named[name]; len(where) > 0 {
				t.Errorf("Fib nomme %s : %s", name, strings.Join(where, ", "))
			}
		}
		for _, name := range algos {
			if where := named[name]; len(where) != 1 {
				t.Errorf("Fib nomme %s %d fois, attendu une seule, dans le retour de son cas : %s", name, len(where), strings.Join(where, ", "))
			}
		}
	})

	for _, name := range algos {
		t.Run(fmt.Sprintf("PERF-GO-11 : le corps de %s ne nomme ni les deux autres algorithmes, ni Fib, ni FibList", name), func(t *testing.T) {
			var forbidden []string
			for _, other := range append([]string{"Fib", "FibList"}, algos...) {
				if other != name {
					forbidden = append(forbidden, other)
				}
			}
			ast.Inspect(funcs[name].Body, func(node ast.Node) bool {
				if id, ok := node.(*ast.Ident); ok && slices.Contains(forbidden, id.Name) {
					t.Errorf("%s : %s nomme %s", fset.Position(id.Pos()), name, id.Name)
				}
				return true
			})
		})
	}
}

// PERF-GO-13, hors catalogue : garde de l'optimisation de la dernière étape de fast (ADR-0004), qui rend F(2k+1) en un
// produit, (2F(k) - F(k-1))(2F(k) + F(k-1)) + 2(-1)^k, et non en deux carrés, F(k)² + (F(k) + F(k-1))²; Rust et Python font
// de même. Les valeurs ne distinguent pas les deux formes, et BIB-05 à 07, BIB-09 et PERF-GO-1 gardent la justesse seule.
// Aucune mesure d'exécution ne les distingue sans fragilité (mesuré sous Go 1.26.0, Linux) : l'empreinte d'allocations de
// fast est la même aux deux formes aux petits n (9 à n = 5, 33 à n = 101) et ne diffère que d'une unité aux grands (51 pour
// le produit contre 52 pour les deux carrés à n = 1023, 111 contre 112 à n = 999 999), donc aucun plafond ne rougit le
// retour aux deux carrés sans dépendre de la réserve que nat.make laisse aux entiers de math/big; les octets alloués ne
// baissent pas à n = 5 ni à n = 101 (288 et 1 248), et de 5 % à n = 1023, de 18 % à n = 999 999, un écart que la version de
// Go, celle de math/big et la taille des blocs déplaceraient. Le chronomètre est exclu. La garde lit donc le texte de
// fib.go par go/parser, comme le volet statique de PERF-GO-11 (l'aide fibFuncs lit le fichier pour les deux), et compte
// les appels de Mul par zone de la dernière étape, les instructions après la boucle : aucun avant le test de parité (les
// instructions entre la boucle et le `if n&1 == 0`, et l'Init de ce `if`, que les deux parités exécutent), exactement un
// dans le bloc du `if`, le produit de F(2k), et exactement un dans la branche impaire, les instructions qui suivent le
// `if` (et son else, s'il en avait un), le produit de F(2k+1). Le bloc pair doit finir par un return, sans quoi ce qui le
// suit n'est pas la branche impaire. La condition se lit dans l'arbre (isParityTest), non dans son texte. Deux comptes
// plus larges laissaient passer des mutants. Un total de deux après la boucle laissait passer une Mul déplacée de la
// branche paire dans une aide, compensée par les deux carrés de la branche impaire. Le reste après la boucle, compté
// comme la branche impaire, laissait passer un produit impair calculé avant le test de parité : calculé puis jeté à chaque
// n pair, sans qu'aucune branche le voie en trop.
//
// Rouge sous : les deux carrés de F(2k+1) (deux Mul dans la branche impaire); la branche paire `return evenStep(a, c)`,
// où l'aide evenStep porte la Mul, avec les deux carrés dans la branche impaire (zéro Mul dans la paire, deux dans
// l'impaire, pour un total de deux); la même aide avec le produit dans la branche impaire (zéro dans la paire); une aide
// qui porte la Mul de la branche impaire (zéro dans l'impaire); un bloc pair qui ne finit pas par un return; le produit
// impair hissé avant le test, `odd := …Mul…` puis `if n&1 == 0 { return … }`, ou dans l'Init du `if`, `var odd *big.Int`
// puis `if odd = …Mul…; n&1 == 0 { … }` (une Mul avant le test de parité, zéro dans la branche impaire); une condition
// mutée, `n&1 != 0` ou `n&1 == 1` avec les branches échangées, `n%2 == 0`, `n&3 == 0`, `1&n == 0` ou `n&1 == 0 && n > 0`
// (plus de `if n&1 == 0` au niveau du corps); un second produit dans un else du `if` (deux dans la branche impaire).
// Reste vert, par intention : le produit impair écrit dans un else du `if n&1 == 0 { return … }`, car c'est lui la branche
// impaire.
// Survivants connus, laissés à la revue de code : la garde ne compte que les appels écrits `x.Mul(…)` dans le corps de
// fast, et ne voit ni ce qu'une branche calcule d'autre ni ce qu'une aide calcule. Une branche qui garde son produit et y
// ajoute un carré par Exp (`x.Exp(x, big.NewInt(2), nil)`) reste verte; une branche qui garde une Mul sans emploi et
// confie les deux carrés à une aide aussi.
func TestPERFGO13_LaDerniereEtapeDeFastCompteUnProduitParBranche(t *testing.T) {
	fset, funcs := fibFuncs(t)
	fn := funcs["fast"]
	if fn == nil {
		t.Fatal("fib.go ne déclare pas la fonction fast : renommée ? Alors adapter PERF-GO-13")
	}
	loop := slices.IndexFunc(fn.Body.List, func(stmt ast.Stmt) bool { _, ok := stmt.(*ast.ForStmt); return ok })
	if loop < 0 {
		t.Fatal("fast n'a plus de boucle for au niveau de son corps : réécrite ? Alors adapter PERF-GO-13")
	}
	last := fn.Body.List[loop+1:] // la dernière étape : les instructions après la boucle
	pos := slices.IndexFunc(last, func(stmt ast.Stmt) bool {
		cond, ok := stmt.(*ast.IfStmt)
		return ok && isParityTest(cond.Cond)
	})
	if pos < 0 {
		t.Fatal("après la boucle, fast n'a plus de `if n&1 == 0 { … }` au niveau de son corps : réécrite ? Alors adapter PERF-GO-13")
	}
	parity := last[pos].(*ast.IfStmt)
	even := parity.Body
	returns := false
	if n := len(even.List); n > 0 {
		_, returns = even.List[n-1].(*ast.ReturnStmt)
	}
	if !returns {
		t.Fatalf("%s : le bloc de la branche paire ne finit pas par un return, donc le reste de fast n'est pas la branche impaire",
			fset.Position(even.Pos()))
	}
	// Trois zones qui ne se recouvrent pas : ce que les deux parités exécutent avant le test, le bloc pair, la branche impaire.
	var before, oddCalls []string
	for _, stmt := range last[:pos] {
		before = append(before, mulCalls(fset, stmt)...)
	}
	if parity.Init != nil {
		before = append(before, mulCalls(fset, parity.Init)...)
	}
	evenCalls := mulCalls(fset, even)
	if parity.Else != nil {
		oddCalls = append(oddCalls, mulCalls(fset, parity.Else)...)
	}
	for _, stmt := range last[pos+1:] {
		oddCalls = append(oddCalls, mulCalls(fset, stmt)...)
	}
	for _, zone := range []struct {
		subject string
		calls   []string
		want    int
		why     string
	}{
		{"le code qui précède le test de parité dans la dernière étape de fast", before, 0,
			"un produit calculé avant le test l'est à chaque n, pair ou impair, et jeté à l'une des deux parités"},
		{"la branche paire de la dernière étape de fast", evenCalls, 1,
			"un produit pour F(2k) = F(k)(F(k) + 2F(k-1)); une Mul déplacée dans une aide la fait tomber à zéro"},
		{"la branche impaire de la dernière étape de fast", oddCalls, 1,
			"un produit pour F(2k+1); les deux carrés en font deux, une Mul déplacée dans une aide zéro"},
	} {
		t.Run(fmt.Sprintf("PERF-GO-13 : %s appelle Mul exactement %d fois", zone.subject, zone.want), func(t *testing.T) {
			if len(zone.calls) != zone.want {
				where := ""
				if len(zone.calls) > 0 {
					where = " (" + strings.Join(zone.calls, ", ") + ")"
				}
				t.Errorf("%s appelle Mul %d fois%s, attendu %d : %s (ADR-0004)",
					zone.subject, len(zone.calls), where, zone.want, zone.why)
			}
		})
	}
}

// mulCalls rend la position de chaque appel d'une méthode Mul sous root.
func mulCalls(fset *token.FileSet, root ast.Node) []string {
	var calls []string
	ast.Inspect(root, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Mul" {
				calls = append(calls, fset.Position(call.Pos()).String())
			}
		}
		return true
	})
	return calls
}

// fibFuncs lit fib.go par go/parser et rend l'ensemble de positions du fichier et ses fonctions sans receveur, par nom : la
// lecture que partagent PERF-GO-11 et PERF-GO-13.
func fibFuncs(t *testing.T) (*token.FileSet, map[string]*ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fib.go", nil, 0) // go test s'exécute dans le dossier du paquet
	if err != nil {
		t.Fatal(err)
	}
	funcs := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
			funcs[fn.Name.Name] = fn
		}
	}
	return fset, funcs
}

// isIdent dit si e est l'identifiant name.
func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// isIntLit dit si e est le littéral entier value, écrit tel quel (« 0 », non « 0x0 »).
func isIntLit(e ast.Expr, value string) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == value
}

// isParityTest dit si e est la condition `n&1 == 0`, lue dans l'arbre : une égalité entre `n & 1` et le littéral 0.
func isParityTest(e ast.Expr) bool {
	eq, ok := e.(*ast.BinaryExpr)
	if !ok || eq.Op != token.EQL || !isIntLit(eq.Y, "0") {
		return false
	}
	and, ok := eq.X.(*ast.BinaryExpr)
	return ok && and.Op == token.AND && isIdent(and.X, "n") && isIntLit(and.Y, "1")
}

// clauseLiteral rend la chaîne que nomme un cas à une seule valeur littérale; ok est faux pour default ou plusieurs valeurs.
func clauseLiteral(clause *ast.CaseClause) (lit *ast.BasicLit, ok bool) {
	if len(clause.List) != 1 {
		return nil, false
	}
	lit, ok = clause.List[0].(*ast.BasicLit)
	return lit, ok && lit.Kind == token.STRING
}

// returnsOwnCall dit si body est l'unique instruction `return <name>(n), nil`.
func returnsOwnCall(body []ast.Stmt, name string) bool {
	if len(body) != 1 {
		return false
	}
	ret, ok := body[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 || !isIdent(ret.Results[1], "nil") {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	return ok && isIdent(call.Fun, name) && len(call.Args) == 1 && isIdent(call.Args[0], "n") && !call.Ellipsis.IsValid()
}
