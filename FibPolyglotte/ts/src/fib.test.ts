// Seam 1 : la bibliothèque. Les attendus viennent de l'oracle (contracts/b000045.txt) ou de faits mathématiques.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { terms } from '../../contracts/oracle.ts';
import { algos, fib, fibList, type Algo } from './fib.ts';

const oracle = (n: number) => BigInt(terms()[n]);

test('BIB-01 : fib(0, iterative) vaut 0', () => {
  assert.equal(fib(0, 'iterative'), oracle(0));
});

test('BIB-02 : fib(1, iterative) vaut 1', () => {
  assert.equal(fib(1, 'iterative'), oracle(1));
});

test('BIB-03 : fib(2, iterative) vaut 1', () => {
  assert.equal(fib(2, 'iterative'), oracle(2));
});

test('BIB-04 : fib(10, iterative) vaut 55', () => {
  assert.equal(fib(10, 'iterative'), oracle(10));
});

// Un sous-test par n : un algorithme faux à n = 5 et à n = 7 rougit les deux, au lieu de s'arrêter au premier échec.
test('BIB-05 : fib(n, iterative) égale l’oracle pour n = 0..2000, au-delà d’int64 et d’uint64', async (t) => {
  for (let n = 0; n <= 2000; n++) await t.test(`n = ${n}`, () => assert.equal(fib(n, 'iterative'), oracle(n)));
});

test('BIB-06 : fib(n, fast) égale l’oracle pour n = 0..2000', async (t) => {
  for (let n = 0; n <= 2000; n++) await t.test(`n = ${n}`, () => assert.equal(fib(n, 'fast'), oracle(n)));
});

test('BIB-07 : fib(n, matrix) égale l’oracle pour n = 0..2000', async (t) => {
  for (let n = 0; n <= 2000; n++) await t.test(`n = ${n}`, () => assert.equal(fib(n, 'matrix'), oracle(n)));
});

const ALGOS: Algo[] = ['iterative', 'matrix', 'fast'];

test('BIB-08 : fib(100000, A) rend le même terme de 20 899 chiffres pour les trois algorithmes', () => {
  const [first, ...others] = ALGOS.map((a) => fib(100000, a));
  for (const t of others) assert.equal(t, first);
  assert.equal(first.toString().length, 20899);
});

for (const algo of ALGOS) {
  test(`BIB-09 : ${algo} respecte l’identité de Cassini à n = 100 000`, () => {
    const [before, at, after] = [99999, 100000, 100001].map((n) => fib(n, algo));
    assert.equal(before * after - at * at, 1n);
  });
}

// La liste rend des chaînes décimales : l'oracle les donne telles quelles.
test('BIB-10 : fibList(0) vaut ["0"]', () => {
  assert.deepEqual(fibList(0), [terms()[0]]);
});

test('BIB-11 : fibList(1) vaut ["0", "1"]', () => {
  assert.deepEqual(fibList(1), [terms()[0], terms()[1]]);
});

test('BIB-12 : fibList(2000) rend les 2 001 termes de l’oracle, dans l’ordre', () => {
  assert.deepEqual(fibList(2000), terms());
});

// hors catalogue : garde du comptage des termes de fibList (boucle à pas de 2, fin paire), pas de l'additionneur.
// Rougit : `if (i < n)` → `if (i <= n)` (un terme de trop quand n est pair) et `i += 2` → `i += 3`.
// Reste vert : la retenue `m = u >> 31` → `m = (u - 1) >> 31`, que gardent BIB-12, PERF-TS-3 à 5, CLI-L05 et CLI-L06;
// et la fin impaire `if (i < n)` → `if (i < n - 1)`, que garde PERF-TS-5 (n = 10000 est pair).
test('PERF-TS-1 : fibList(10000) rend 10 001 termes', () => {
  assert.equal(fibList(10000).length, 10001);
});

// hors catalogue : garde du limbe de poids fort du dernier terme (retenue qui sort, zéros de tête retirés).
// Rougit : `if (m === 0)` → `if (m === 1)` (la retenue qui sort du dernier limbe est perdue) et la suppression de
// `while (bytes[start] === 0x30) start++`. Pas l'additionneur : la retenue `m = u >> 31` → `m = (u - 1) >> 31` la
// laisse verte.
test('PERF-TS-2 : le dernier terme de fibList(10000) compte 2 090 chiffres', () => {
  assert.equal(fibList(10000)[10000].length, 2090);
});

// hors catalogue : garde de l'optimisation de la liste (écriture ASCII des limbes, zéros de tête retirés)
test('PERF-TS-3 : chaque terme de fibList(10000) s’écrit sans zéro de tête', () => {
  fibList(10000).forEach((term, i) => assert.match(term, /^(0|[1-9][0-9]*)$/, `n = ${i}`));
});

// hors catalogue : garde de l'optimisation de la liste (retenue propagée entre limbes de 4 chiffres)
test('PERF-TS-4 : chaque terme de fibList(10000) dès F(2) est la somme des deux précédents', () => {
  const list = fibList(10000);
  for (let i = 2; i < list.length; i++) {
    assert.equal(BigInt(list[i]), BigInt(list[i - 1]) + BigInt(list[i - 2]), `n = ${i}`);
  }
});

// hors catalogue : garde de la taille du tampon aux petits n et de la fin impaire de la boucle. Dans ts/src, seul ce
// test rougit : `Math.ceil((n * 0.209) / 4) + 1` → `Math.round((n * 0.209) / 4)` (un limbe de trop peu dès n = 2) et
// `if (i < n)` → `if (i < n - 1)`. BIB-12 et PERF-TS-1 à 4 n'éprouvent que n = 2000 et n = 10000 : pairs, et où round
// et ceil donnent le même nombre de limbes. `Math.round` avec le « + 1 » conservé est équivalent : le limbe
// d'assurance absorbe l'arrondi.
test('PERF-TS-5 : fibList(n) rend les n + 1 premiers termes de l’oracle pour chaque n de 0 à 300', () => {
  const expected = terms();
  for (let n = 0; n <= 300; n++) assert.deepEqual(fibList(n), expected.slice(0, n + 1), `n = ${n}`);
});

for (const bogus of ['all', 'bogus', '', 'FAST', 'toString']) {
  test(`BIB-13 : fib(5, ${JSON.stringify(bogus)}) est refusé`, () => {
    assert.throws(() => fib(5, bogus as Algo), { name: 'TypeError', message: /algorithme inconnu/ });
  });
}

// Un test par indice refusé : un contrôle qui laisse passer -1 et NaN rougit les deux, pas seulement -1.
const BAD_INDEXES = [-1, 1.5, NaN];

for (const algo of ALGOS) {
  for (const n of BAD_INDEXES) {
    test(`BIB-14 : fib refuse ${n} pour ${algo}`, () => {
      assert.throws(() => fib(n, algo), { name: 'RangeError', message: /indice invalide/ });
    });
  }
}

for (const n of BAD_INDEXES) {
  test(`BIB-14 : fibList refuse ${n}`, () => {
    assert.throws(() => fibList(n), { name: 'RangeError', message: /indice invalide/ });
  });
}

// hors catalogue : garde d'identité de la table `algos` (F2, F6; lacune de PERF-TS-8, qui espionne le nom demandé, pas la
// fonction rangée derrière lui). Les trois algorithmes rendent les mêmes valeurs : aucun cas de valeur ne voit un nom
// aiguillé ailleurs. Ni valeur calculée ni durée : le nom de la fonction rangée sous chaque clé, puis le début de sa source
// (une déclaration), puis l'entrée que `fib` appelle, par des espions dans la table. L'export `algos` a deux usages : les
// tests du nom la lisent, les espions y écrivent. Il est en lecture seule par le type, et seuls les espions le convertissent
// en `Record` pour écrire. Un `Object.freeze` de la table les ferait rougir à tort, leur affectation lèverait une
// TypeError, sans que `fib` ait changé.
// Rougit, par le test du nom aiguillé ailleurs, les six échanges de la table : `{ iterative: matrix, matrix, fast }`,
// `{ iterative: fast, matrix, fast }`, `{ iterative, matrix: iterative, fast }`, `{ iterative, matrix: fast, fast }`,
// `{ iterative, matrix, fast: iterative }` et `{ iterative, matrix, fast: matrix }`. Le nom seul laisse passer une
// enveloppe, car dans un littéral d'objet une fonction anonyme prend le nom de sa clé : `matrix: (n: number) => fast(n)`,
// `matrix(n: number) { return fast(n); }` et `matrix: function (n: number) { return fast(n); }`. Le début de la source,
// `function matrix(`, les rougit toutes les trois (la compilation de tsconfig.build.json l'émet de même). L'expression de
// fonction nommée du même nom, `matrix: function matrix(n: number) { return fast(n); }`, garde le nom et le début de la
// source : le test du corps, plus bas, la rougit.
for (const algo of ALGOS) {
  test(`PERF-TS-10 : la table algos range la fonction ${algo} sous le nom ${algo}`, () => {
    assert.equal(algos[algo].name, algo);
    assert.ok(algos[algo].toString().startsWith(`function ${algo}(`));
  });
}

// Rougit, par le test des deux autres noms, l'aiguillage de `fib` fixé à un nom : `algos[algo](n)` → `algos.fast(n)`,
// `algos.matrix(n)` ou `algos.iterative(n)`. Le 0n des espions n'est comparé à rien.
// Les espions jouent aussi n = 1000000 (la borne F8; ils ne calculent rien, donc le coût est nul) : seul ce second indice rougit un
// aiguillage qui dépend de n, `return algos[n > 10 && algo === 'matrix' ? 'fast' : algo](n);` dans `fib`, que n = 3 laisse passer.
// Survivant connu, laissé à la revue de code : un aiguillage qui dépend de n hors de ces deux indices, tel que
// `return algos[n > 10 && n < 1000000 && algo === 'matrix' ? 'fast' : algo](n);`.
const SPIED_INDEXES = [3, 1000000];

for (const n of SPIED_INDEXES) {
  for (const algo of ALGOS) {
    test(`PERF-TS-10 : fib(${n}, ${algo}) appelle l’entrée ${algo} de la table, et elle seule`, () => {
      // Conversion explicite : `algos` est en lecture seule par le type; seuls les espions (PERF-TS-10, PERF-TS-12) y écrivent, et chacun restaure la table dans finally.
      const table = algos as Record<Algo, (n: number) => bigint>;
      const original = { ...table };
      const called: Algo[] = [];
      try {
        for (const name of ALGOS) {
          table[name] = () => {
            called.push(name);
            return 0n;
          };
        }
        fib(n, algo);
      } finally {
        Object.assign(table, original);
      }
      assert.deepEqual(called, [algo]);
    });
  }
}

// hors catalogue : garde du corps de chaque entrée de `algos` (F2; lacune de la garde d'identité ci-dessus, qui lit le nom et le
// début de la source, et des espions, qui ne voient que l'aiguillage de `fib`). Un nom qui calcule par un autre algorithme rend les
// mêmes valeurs, donc aucun cas de valeur ne le voit, et une expression de fonction nommée du même nom garde le début de la
// source. Le texte brut de chaque entrée, `algos[algo].toString()`, ne nomme donc ni l'un des deux autres algorithmes, ni `fib`,
// ni `fibList`, ni `algos`, ni `this`; son propre nom, celui de sa déclaration, est permis. `this` est interdit parce que `fib`
// appelle `algos[algo](n)` en méthode : `this` vaut `algos` dans chaque entrée, et `Object.values(this)[2](n)` délègue à `fast`
// sans nommer personne.
// Le texte est lu brut, sans retirer les commentaires : ni un commentaire, ni un littéral (chaîne, gabarit, regex) ne peut masquer
// un nom écrit dans le texte. Ce qui reste hors de portée est ce que le texte ne nomme pas : la délégation par une fonction
// auxiliaire ou par un nom calculé (survivants ci-dessous), et la liste de ces survivants n'est pas exhaustive. Un retrait des
// commentaires par regex a été essayé et abandonné : chaque revue y trouvait un survivant (une continuation de ligne dans une
// chaîne, un commentaire collé au code qui ouvre un faux bloc ou une fausse chaîne, une fin de ligne en CR, U+2028 ou U+2029, un
// littéral regex tel que `/[ //]/` ou `/\/*/`).
// Rougit : `return fast(n);` en tête de `matrix`; `const re = /\/\//; return fast(n);`; `const re = /[ //]/; return fast(n);`;
// `const re = /\/*/; return fast(n); const end = /a*/;`; une continuation de ligne dans une chaîne (`const s = "a\`, un saut de
// ligne, puis ` //"; if (s) return fast(n);`); un commentaire collé au code (`let q = 1;// voir /* ici`, un saut de ligne, puis
// `if (q) return fast(n); /* fin */`); un commentaire `//` clos par un saut de ligne CR, U+2028 ou U+2029, suivi de
// `return fast(n);` (ces trois caractères sont du vrai code après le commentaire; le fichier les désigne ici par leur nom et ne
// les contient pas); la délégation par `this`, qui passe tsc et toute la suite de valeurs :
// `function matrix(this: unknown, n: number): bigint { if (n > 0) return Object.values(this as object)[2](n); …`;
// `{ iterative, matrix: function matrix(n: number) { return fast(n); }, fast }` dans la table; `return fib(n, 'fast');`;
// et `if (n < 64) return iterative(n);` en tête de `fast`.
// Ce dernier rouge est voulu, non une erreur du test : F2 veut que chaque nom calcule par son propre algorithme à tout n, pas
// seulement au-dessus d'un seuil. Un travail de performance qui aiguillerait les petits indices vers un autre algorithme
// rougit ici avec raison; il doit d'abord changer F2 au PRD, puis ce test, au lieu d'assouplir le test seul.
// Le prix du texte brut est un faux rouge, bruyant, jamais silencieux : un commentaire de l'un de ces corps qui nomme un autre
// algorithme (`// plus lent que fast` dans `matrix`, ou `let q = 1; // note if (q) return fast(n);`, où toute la fin de la ligne
// est un vrai commentaire et où rien ne délègue) fait rougir la garde, de même qu'un commentaire qui contient `this`. Il faut
// alors écrire le commentaire autrement, « le doublement rapide » au lieu de `fast`, et non revenir à un retrait des commentaires.
// Survivants connus, laissés à la revue de code : une délégation par une fonction auxiliaire (`return viaFast(n);`, où `viaFast`,
// définie hors de la table, rend `fast(n)`) ou par un nom calculé (`return pick('fa' + 'st')(n);`, où `pick` choisit dans un
// objet défini hors de la table); le texte de l'entrée ne nomme alors aucun algorithme. Cette liste n'est pas exhaustive.
const FORBIDDEN = [...ALGOS, 'fib', 'fibList', 'algos', 'this'];

for (const algo of ALGOS) {
  test(`PERF-TS-10 : le corps de ${algo} ne nomme ni un autre algorithme, ni fib, ni fibList, ni algos, ni this`, () => {
    const source = algos[algo].toString();
    const named = FORBIDDEN.filter((name) => name !== algo && new RegExp(`\\b${name}\\b`, 'u').test(source));
    assert.deepEqual(named, [], source);
  });
}
