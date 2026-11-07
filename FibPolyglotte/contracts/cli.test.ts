// Test de contrat : vérifie l'oracle, puis joue les cas CLI de docs/cas-de-test.md sur chaque implémentation.
// Usage : node contracts/cli.test.ts [go|rust|ts|py …]
import { before, describe, test } from 'node:test';
import assert from 'node:assert/strict';
import cp, { spawnSync } from 'node:child_process';
import fs, { existsSync, lstatSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, statSync, symlinkSync, writeFileSync } from 'node:fs';
import { syncBuiltinESMExports } from 'node:module';
import { tmpdir } from 'node:os';
import { dirname, join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { abPairs, abRound, main, measure, pair, paired, parseArgs, rounds } from './bench.ts';
import { build, closeStderr, command, IDS, launch, present, ROOT, type LaunchOptions } from './lancement.ts';
import { CHECKS, lines, terms } from './oracle.ts';

const USAGE = `usage: fib [--algo iterative|matrix|fast|all] [--time] N
       fib --list [--time] N
       fib --help
`;

// 'erreur' : code 2, stdout vide, une seule ligne `fib: …`. Sinon code 0 : `out` exact et/ou `check`, `times` = algos chronométrés.
type Want = 'erreur' | { out?: string | (() => string); check?: (out: string) => void; times?: string[] };
type Case = [id: string, title: string, args: string[], want: Want, env?: LaunchOptions['env']];

// term et list sont paresseux : un oracle absent ne fait échouer que les cas qui en tirent leur attendu (catalogue § 3).
const term = (k: number) => () => `${terms()[k]}\n`;
const list = (n: number) => () => terms().slice(0, n + 1).map((t) => `${t}\n`).join('');

type Run = ReturnType<typeof launch>;

const assertAscii = (r: Run) =>
  assert.ok(/^[\n\x20-\x7e]*$/.test(r.out + r.err), 'X-4 : sortie non ASCII, ou avec \\r ou ESC');

// Assertions communes X-1 à X-4.
function verify(r: Run, want: Want): void {
  const ctx = `code ${r.code}, stderr ${JSON.stringify(r.err.slice(0, 200))}`;
  if (want === 'erreur') {
    assert.equal(r.code, 2, `X-1 : ${ctx}`);
    assert.match(r.err, /^fib: [^\n]*\n$/, 'X-3 : une seule ligne « fib: … »');
    assert.equal(r.out, '', 'X-2 : stdout vide');
    return assertAscii(r);
  }
  assert.equal(r.code, 0, `X-1 : ${ctx}`);
  const got = r.err.split('\n');
  assert.equal(got.pop(), '', `X-3 : stderr se termine par \\n (${ctx})`);
  got.forEach((l) => assert.match(l, /^(fast|iterative|matrix) [0-9]+\.[0-9]{3} ms$/, 'X-3 : ligne de durée'));
  assert.deepEqual(got.map((l) => l.split(' ')[0]), want.times ?? [], 'X-3 : algorithmes chronométrés');
  assertAscii(r);
  const exact = typeof want.out === 'function' ? want.out() : want.out;
  if (exact !== undefined) assert.equal(r.out, exact, 'X-2');
  want.check?.(r.out);
}

// Partagé par CLI-T12 et CLI-T13 : F(k) divise F(mk), donc F(2000) divise F(100 000) et F(1 000 000) (catalogue § 2).
// F(2000) vient de l'oracle. Reste lu par tranches de 18 chiffres, sans la conversion quadratique décimal → BigInt.
const checkDivisibleByF2000 = (out: string) => {
  const f = BigInt(terms()[2000]);
  const digits = out.slice(0, -1);
  const padded = digits.padStart(Math.ceil(digits.length / 18) * 18, '0');
  const tranche = 10n ** 18n;
  let r = 0n;
  for (let i = 0; i < padded.length; i += 18) r = (r * tranche + BigInt(padded.slice(i, i + 18))) % f;
  assert.ok(r === 0n, 'la sortie n’est pas divisible par F(2000)');
};

const T: Case[] = [
  ['CLI-T01', '10 écrit F(10)', ['10'], { out: term(10) }],
  ['CLI-T02', '0 écrit F(0)', ['0'], { out: term(0) }],
  ['CLI-T03', '1 écrit F(1)', ['1'], { out: term(1) }],
  ['CLI-T04', '2 écrit F(2)', ['2'], { out: term(2) }],
  ['CLI-T05', '92 écrit F(92), dernier terme d’un int64', ['92'], { out: term(92) }],
  ['CLI-T06', '93 écrit F(93), dernier terme d’un uint64', ['93'], { out: term(93) }],
  ['CLI-T07', '94 écrit F(94), au-delà d’un uint64', ['94'], { out: term(94) }],
  ['CLI-T08', '1000 écrit F(1000)', ['1000'], { out: term(1000) }],
  ['CLI-T09', '2000 écrit F(2000)', ['2000'], { out: term(2000) }],
  ['CLI-T10', 'des zéros de tête sont permis : 007 vaut 7', ['007'], { out: term(7) }],
  ['CLI-T11', 'de nombreux zéros de tête : …010 vaut 10', ['00000000000000000000000000010'], { out: term(10) }],
  ['CLI-T12', '100000 écrit un terme de 20 899 chiffres, congru à F(1696) mod 2048 et à F(1000) mod 1000, divisible par F(2000)', ['100000'], {
    check: (out) => {
      assert.match(out, /^[1-9][0-9]*\n$/);
      assert.equal(out.length - 1, 20899);
      const t = BigInt(out);
      assert.equal(t % 2048n, BigInt(terms()[1696]) % 2048n);
      assert.equal(t % 1000n, BigInt(terms()[1000]) % 1000n);
      checkDivisibleByF2000(out);
    },
  }],
  ['CLI-T13', '1000000 écrit un terme de 208 988 chiffres, congru à F(1600) mod 2048 et à F(1000) mod 1000, divisible par F(2000)', ['1000000'], {
    check: (out) => {
      assert.match(out, /^[1-9][0-9]*\n$/);
      assert.equal(out.length - 1, 208988);
      // Restes lus chiffre à chiffre : évite la conversion décimal → BigInt, quadratique, de 208 988 chiffres.
      const mod = (m: number) => [...out.slice(0, -1)].reduce((r, c) => (r * 10 + Number(c)) % m, 0);
      assert.equal(mod(2048), Number(BigInt(terms()[1600]) % 2048n));
      assert.equal(mod(1000), Number(BigInt(terms()[1000]) % 1000n));
      checkDivisibleByF2000(out);
    },
  }],
  ['CLI-T14', '5 000 zéros de tête, puis 5, valent 5', ['0'.repeat(5000) + '5'], { out: term(5) }],
];

const I: Case[] = [
  ['CLI-I01', 'aucun argument : indice absent', [], 'erreur'],
  ['CLI-I02', 'un argument vide est refusé', [''], 'erreur'],
  ['CLI-I03', 'abc n’est pas un nombre', ['abc'], 'erreur'],
  ['CLI-I04', 'un décimal est refusé', ['1.5'], 'erreur'],
  ['CLI-I05', 'un signe + est refusé', ['+5'], 'erreur'],
  ['CLI-I06', 'un soulignement est refusé', ['1_000'], 'erreur'],
  ['CLI-I07', 'un hexadécimal est refusé', ['0x10'], 'erreur'],
  ['CLI-I08', 'une espace de tête est refusée', [' 5'], 'erreur'],
  ['CLI-I09', 'une espace de queue est refusée', ['5 '], 'erreur'],
  ['CLI-I10', 'un chiffre non ASCII est refusé', ['٣'], 'erreur'],
  ['CLI-I11', '-1 est une option inconnue', ['-1'], 'erreur'],
  ['CLI-I12', '1000001 dépasse la borne d’un terme', ['1000001'], 'erreur'],
  ['CLI-I13', 'la borne s’applique malgré les zéros de tête', ['000000000000000000000000001000001'], 'erreur'],
  ['CLI-I14', 'un indice trop grand pour un entier natif est une erreur, pas une panique', ['99999999999999999999'], 'erreur'],
  ['CLI-I15', 'deux indices sont refusés', ['5', '6'], 'erreur'],
  ['CLI-I16', 'un saut de ligne après l’indice est refusé', ['5\n'], 'erreur'],
  ['CLI-I17', 'une espace entre deux nombres, dans un seul argument, est refusée', ['5 6'], 'erreur'],
];

const A: Case[] = [
  ['CLI-A01', '--algo iterative écrit F(2000)', ['--algo', 'iterative', '2000'], { out: term(2000) }],
  ['CLI-A02', '--algo matrix écrit F(2000)', ['--algo', 'matrix', '2000'], { out: term(2000) }],
  ['CLI-A03', '--algo fast écrit F(2000)', ['--algo', 'fast', '2000'], { out: term(2000) }],
  ['CLI-A04', '--algo all écrit F(2000) une seule fois', ['--algo', 'all', '2000'], { out: term(2000) }],
  ['CLI-A05', '--algo all 0 écrit F(0)', ['--algo', 'all', '0'], { out: term(0) }],
  ['CLI-A06', '--algo=matrix équivaut à --algo matrix', ['--algo=matrix', '2000'], { out: term(2000) }],
  ['CLI-A07', 'l’option peut suivre l’indice', ['2000', '--algo', 'matrix'], { out: term(2000) }],
  ['CLI-A08', 'répéter --algo est accepté', ['--algo', 'fast', '--algo', 'matrix', '2000'], { out: term(2000) }],
  ['CLI-A09', '--algo sans valeur est refusé', ['--algo'], 'erreur'],
  ['CLI-A10', '--algo= avec une valeur vide est refusé', ['--algo=', '5'], 'erreur'],
  ['CLI-A11', 'un algorithme inconnu est refusé', ['--algo', 'bogus', '5'], 'erreur'],
  ['CLI-A12', 'le nom de l’algorithme est sensible à la casse', ['--algo', 'FAST', '5'], 'erreur'],
  ['CLI-A13', '--algo 5 prend 5 pour valeur, et l’indice manque', ['--algo', '5'], 'erreur'],
  ['CLI-A14', 'le nom de l’option est sensible à la casse', ['--Algo', 'fast', '5'], 'erreur'],
  ['CLI-A15', '--algo --list 5 prend --list pour valeur', ['--algo', '--list', '5'], 'erreur'],
  ['CLI-A16', 'chaque occurrence de --algo doit être valide, même si une valide suit', ['--algo', 'bogus', '--algo', 'fast', '5'], 'erreur'],
  ['CLI-A17', '--algo avec une valeur vide (forme séparée) est refusé', ['--algo', '', '5'], 'erreur'],
  ['CLI-A18', 'un saut de ligne dans un argument ne produit pas deux lignes', ['--algo', 'x\ny', '5'], 'erreur'],
  ['CLI-A19', 'la casse compte à chaque occurrence de --algo, même si une valide suit', ['--algo', 'FAST', '--algo', 'fast', '5'], 'erreur'],
  ['CLI-A20', 'une valeur non ASCII de --algo : la ligne d’erreur reste en ASCII', ['--algo', 'fäst', '5'], 'erreur'],
];

// Partagé par CLI-L06 et CLI-L16 : la liste de 10 001 termes, quelle que soit l'écriture de l'indice.
const checkList10000 = (out: string) => {
  assert.ok(out.endsWith('\n'), 'la dernière ligne se termine par \\n');
  const rows = out.split('\n');
  rows.pop();
  assert.equal(rows.length, 10001);
  assert.deepEqual(rows.slice(0, 2001), terms());
  for (let i = 2; i < rows.length; i++) {
    // BigInt() accepte `007`, ` 12`, `12 `, `+5` et `0x10` : la forme décimale se vérifie avant la somme.
    assert.match(rows[i], /^(0|[1-9][0-9]*)$/, `ligne ${i}`);
    assert.equal(BigInt(rows[i]), BigInt(rows[i - 1]) + BigInt(rows[i - 2]), `ligne ${i}`);
  }
  assert.equal(rows[10000].length, 2090);
};

const L: Case[] = [
  ['CLI-L01', '--list 0 écrit seulement F(0)', ['--list', '0'], { out: list(0) }],
  ['CLI-L02', '--list 1 écrit F(0) et F(1)', ['--list', '1'], { out: list(1) }],
  ['CLI-L03', '--list 2 écrit trois lignes', ['--list', '2'], { out: list(2) }],
  ['CLI-L04', '--list 20 écrit les 21 premiers termes', ['--list', '20'], { out: list(20) }],
  ['CLI-L05', '--list 2000 écrit les 2 001 termes de l’oracle', ['--list', '2000'], { out: list(2000) }],
  ['CLI-L06', '--list 10000 écrit 10 001 lignes cohérentes avec l’oracle et la récurrence', ['--list', '10000'], { check: checkList10000 }],
  ['CLI-L07', 'le drapeau peut suivre l’indice', ['5', '--list'], { out: list(5) }],
  ['CLI-L08', 'répéter --list ne change rien', ['--list', '--list', '3'], { out: list(3) }],
  ['CLI-L09', '10001 dépasse la borne d’une liste', ['--list', '10001'], 'erreur'],
  ['CLI-L10', '--list et --algo s’excluent', ['--list', '--algo', 'fast', '5'], 'erreur'],
  ['CLI-L11', '--algo et --list s’excluent dans l’autre ordre', ['--algo', 'fast', '--list', '5'], 'erreur'],
  ['CLI-L12', '--list et --algo all s’excluent', ['--list', '--algo', 'all', '5'], 'erreur'],
  ['CLI-L13', '--list sans indice est refusé', ['--list'], 'erreur'],
  ['CLI-L14', '--list=5 est refusé : un drapeau ne prend pas de valeur', ['--list=5'], 'erreur'],
  ['CLI-L15', 'la borne d’une liste s’applique malgré les zéros de tête', ['--list', '00010001'], 'erreur'],
  ['CLI-L16', 'des zéros de tête sont permis pour une liste : 010000 vaut 10000', ['--list', '010000'], { check: checkList10000 }],
];

const D: Case[] = [
  ['CLI-D01', '--time écrit la durée de fast sur stderr', ['--time', '50'], { out: term(50), times: ['fast'] }],
  ['CLI-D02', '--time avec --algo all chronomètre fast, iterative puis matrix', ['--time', '--algo', 'all', '50'], { out: term(50), times: ['fast', 'iterative', 'matrix'] }],
  ['CLI-D03', '--time avec --algo matrix chronomètre matrix', ['--time', '--algo', 'matrix', '50'], { out: term(50), times: ['matrix'] }],
  ['CLI-D04', 'la dernière occurrence de --algo gagne', ['--time', '--algo', 'fast', '--algo', 'matrix', '50'], { out: term(50), times: ['matrix'] }],
  ['CLI-D05', 'la dernière occurrence de --algo gagne aussi sur all', ['--algo', 'all', '--algo', 'iterative', '--time', '50'], { out: term(50), times: ['iterative'] }],
  ['CLI-D06', '--time avec --list chronomètre iterative', ['--time', '--list', '5'], { out: list(5), times: ['iterative'] }],
  ['CLI-D07', '--time peut suivre l’indice', ['50', '--time'], { out: term(50), times: ['fast'] }],
  ['CLI-D08', 'répéter --time ne donne qu’une ligne', ['--time', '--time', '50'], { out: term(50), times: ['fast'] }],
  ['CLI-D09', '--time sans indice est refusé', ['--time'], 'erreur'],
  ['CLI-D10', '--time=1 est refusé : un drapeau ne prend pas de valeur', ['--time=1', '5'], 'erreur'],
  ['CLI-D11', 'un indice invalide ne laisse aucune ligne de durée', ['--time', 'abc'], 'erreur'],
];

const H: Case[] = [
  ['CLI-H01', '--help écrit l’usage sur stdout, code 0', ['--help'], { out: USAGE }],
  ['CLI-H02', '-h écrit l’usage', ['-h'], { out: USAGE }],
  ['CLI-H03', 'l’aide l’emporte sur une option inconnue', ['--help', '--bogus'], { out: USAGE }],
  ['CLI-H04', 'l’aide l’emporte où qu’elle soit', ['5', '-h'], { out: USAGE }],
  ['CLI-H05', 'l’aide l’emporte à la place de la valeur de --algo', ['--algo', '--help'], { out: USAGE }],
  ['CLI-H06', 'l’aide l’emporte sur une borne dépassée', ['--list', '10001', '--help'], { out: USAGE }],
  ['CLI-H07', 'répéter --help écrit l’usage une fois', ['--help', '--help'], { out: USAGE }],
  ['CLI-H08', '-help est une option inconnue', ['-help'], 'erreur'],
  ['CLI-H09', '--HELP est refusé : le nom de l’option est sensible à la casse', ['--HELP'], 'erreur'],
  ['CLI-H10', '--algo=--help n’est pas l’aide', ['--algo=--help', '5'], 'erreur'],
  ['CLI-H11', 'l’aide l’emporte sur un indice invalide', ['abc', '--help'], { out: USAGE }],
];

const U: Case[] = [
  ['CLI-U01', 'une option longue inconnue est refusée', ['--bogus', '5'], 'erreur'],
  ['CLI-U02', 'un tiret seul est une option inconnue', ['-', '5'], 'erreur'],
  ['CLI-U03', 'un double tiret est une option inconnue, pas une fin des options', ['--', '5'], 'erreur'],
  ['CLI-U04', 'option non ASCII : la ligne d’erreur reste en ASCII', ['--bogüs', '5'], 'erreur'],
  ['CLI-U05', 'plusieurs caractères non ASCII dans une option : la ligne d’erreur reste en ASCII', ['--bögüs', '5'], 'erreur'],
  ['CLI-U06', 'deux fautes ne produisent qu’une seule ligne fib:', ['--bogus', 'abc'], 'erreur'],
];

const V: Case[] = [
  ['CLI-V01', 'la locale ne change pas le point décimal des durées', ['--time', '50'], { out: term(50), times: ['fast'] },
    { LANG: 'fr_CA.UTF-8', LC_ALL: 'fr_CA.UTF-8', LC_NUMERIC: 'fr_CA.UTF-8' }],
  ['CLI-V03', 'PYTHONIOENCODING ne change pas les octets écrits', ['--time', '50'], { out: term(50), times: ['fast'] }, { PYTHONIOENCODING: 'utf-16' }],
];

// CLI-V02 joue trois exécutions (U01, H01 puis T01) : l'oracle ne sert qu'à la dernière, pour que l'ESC soit cherché partout.
const V02 = (impl: string) => {
  // NO_COLOR retiré (undefined : spawnSync l'omet), non vidé. Hérité du parent, il l'emporterait sur FORCE_COLOR sous Python 3.14 et viderait le cas de son sens.
  // Vidé, il ferait écrire à Node 22 un avertissement sur stderr dès qu'un console.* tourne avec FORCE_COLOR, et rougir X-3 pour une raison du harnais.
  const env = { FORCE_COLOR: '1', CLICOLOR_FORCE: '1', PYTHON_COLORS: '1', NO_COLOR: undefined };
  verify(launch(impl, ['--bogus', '5'], { env }), 'erreur');
  verify(launch(impl, ['--help'], { env }), { out: USAGE });
  verify(launch(impl, ['10'], { env }), { out: term(10) });
};

// Sous Windows, ni Node ni le contrat ne savent lancer un processus sans stderr : la raison s'affiche à la place du test.
const noShell = process.platform === 'win32' && 'Windows : ni Node ni le contrat ne lancent un processus sans stderr';

// CLI-V04 joue trois exécutions, le fd 2 fermé : sans stderr, X-3 ne s'applique pas, et `verify` ne sert pas.
// L'oracle ne sert qu'aux deux dernières, après le code. X-4 n'a pas d'assertion à part : l'égalité exacte de stdout, dont
// les attendus sont en ASCII, le couvre, et stderr vide ne laisse rien à y chercher.
const V04 = (impl: string) => {
  const runs: [args: string[], code: number, out: () => string][] = [
    [['--bogus', '5'], 2, () => ''],
    [['--time', '50'], 0, term(50)],
    [['10'], 0, term(10)],
  ];
  for (const [args, code, out] of runs) {
    const r = launch(impl, args, { stderrClosed: true });
    // Si le fd 2 restait ouvert, la ligne `fib: …` ou la durée s'y lirait : stderr vide prouve que `launch` a fermé le fd 2 pour la CLI.
    assert.equal(r.err, '', `le fd 2 n'est pas fermé : fib ${args.join(' ')}, stderr ${JSON.stringify(r.err.slice(0, 200))}`);
    assert.equal(r.code, code, `X-1 : fib ${args.join(' ')}, code ${r.code}`);
    assert.equal(r.out, out(), `X-2 : fib ${args.join(' ')}`);
  }
};

const CASES: Case[] = [...T, ...I, ...A, ...L, ...D, ...H, ...U, ...V];

describe('oracle', () => {
  for (const [id, title, check] of CHECKS) test(`${id} : ${title}`, () => check(lines()));
});

// Les contrôles du harnais, joués une fois pour les quatre CLI : ils éprouvent le test de contrat lui-même et rougissent
// quand il ferait passer un cas à vide. La sonde du fd 2 lance un vrai processus; HAR-1 à HAR-3 et HAR-5 à HAR-7 n'en lancent aucun.
describe('harnais', () => {
  // Si `2>&-` cessait de fermer le fd 2, CLI-V04 passerait à vide. Une redirection vers un fd fermé échoue (code non nul).
  test('CLI-V04 : le harnais ferme le fd 2', { skip: noShell }, () => {
    const [shell, shellArgs] = closeStderr(['/bin/sh', '-c', 'echo x >&2; echo $?']);
    const probe = spawnSync(shell, shellArgs, { encoding: 'latin1' });
    assert.ifError(probe.error);
    assert.match(probe.stdout, /^[1-9][0-9]*\n$/, 'le harnais ne ferme pas le fd 2');
  });

  // HAR-1 à HAR-3 et HAR-5 remplacent spawnSync par un espion qui note l'appel et rend une exécution factice : rien n'est lancé ni
  // construit, et ils valent sous Windows.
  type Call = { file: string; args: readonly string[]; options: { cwd?: string; env?: NodeJS.ProcessEnv } };
  // `reply` change la réponse factice d'un appel (un code non nul, une sortie précise); par défaut, une exécution réussie.
  type Reply = { stdout: string; stderr: string; status: number | null };
  const spy = (f: () => unknown, reply?: (call: Call) => Partial<Reply>): Call[] => {
    const real = cp.spawnSync;
    const calls: Call[] = [];
    const fake = (file: string, args: readonly string[], options: Call['options']) => {
      const call = { file, args, options };
      calls.push(call);
      return { pid: 0, output: [], stdout: '55\n', stderr: 'fast 0.010 ms\n', status: 0, signal: null, ...reply?.(call) };
    };
    Object.assign(cp, { spawnSync: fake });
    syncBuiltinESMExports();
    try {
      f();
    } finally {
      Object.assign(cp, { spawnSync: real });
      syncBuiltinESMExports();
    }
    return calls;
  };

  // HAR-1 : le banc lance exactement la commande de l'implémentation, suivie de --time et des arguments : une propriété, non une
  // liste noire de shells. Tout enveloppeur (sh, env, ksh -c, nice, time…) ajouterait son démarrage à chaque mesure sans changer
  // aucune sortie. `command(id)` vient du tableau de lancement, non de `launch` : un défaut de `launch` ne fausse pas l'attendu.
  // Le second contrôle ajoute les options (cwd, env, tampon, délai), identiques à celles d'un lancement du contrat.
  // Ensemble : present(), les implémentations que le contrat et le banc jouent par défaut (HAR-3 prend tous les identifiants).
  test('HAR-1 : le banc lance la commande de chaque CLI, suivie de --time et des arguments, sans enveloppeur', () => {
    for (const id of present()) {
      const bench = spy(() => measure(id, ['10']));
      assert.equal(bench.length, 1, `${id} : une mesure, un lancement`);
      assert.deepEqual([bench[0].file, ...bench[0].args], [...command(id), '--time', '10'], `${id} : le banc ne lance pas la commande de l'implémentation`);
      assert.deepEqual(bench, spy(() => launch(id, ['--time', '10'])), `${id} : le banc ne lance pas l'appel du contrat`);
    }
  });

  // HAR-2 : l'env d'un cas s'ajoute à celui du processus, et undefined retire la variable (NO_COLOR de CLI-V02). Les CLI
  // ignorent ces variables par contrat : sur a906e27, un env perdu, substitué ou écrasé laissait le contrat vert (403/403).
  // La seule variable que le contrôle écrit dans process.env est FIB_RETIREE : il la rend telle qu'il l'a trouvée, absente ou non.
  test('HAR-2 : l’env d’un cas s’ajoute à celui du processus, et undefined retire la variable', () => {
    const avant = process.env.FIB_RETIREE;
    process.env.FIB_RETIREE = 'r';
    try {
      const [{ options }] = spy(() => launch('py', ['10'], { env: { FIB_SONDE: 'x', FIB_RETIREE: undefined } }));
      assert.equal(options.env?.FIB_SONDE, 'x', 'l’env du cas ne parvient pas au fils');
      assert.equal(options.env?.FIB_RETIREE, undefined, 'une variable mise à undefined reste dans l’env du fils');
      for (const k of Object.keys(process.env)) if (k !== 'FIB_RETIREE') assert.equal(options.env?.[k], process.env[k], `${k} n’est pas hérité`);
    } finally {
      if (avant === undefined) delete process.env.FIB_RETIREE;
      else process.env.FIB_RETIREE = avant;
    }
  });

  // HAR-3 : les CLI passent les mêmes cas, donc le contrat ne voit pas laquelle il lance. Sur a906e27, un `run` qui lançait
  // toujours Python laissait 403/403 : chaque identifiant doit lancer sa propre commande.
  // Ensemble : IDS, tous les identifiants du PRD § 5, présents ou non : `launch` ne lit pas le disque et l'espion ne lance rien.
  test('HAR-3 : chaque identifiant lance sa propre commande', () => {
    const cmds = IDS.map((id) => spy(() => launch(id, ['10'])).map((c) => [c.file, ...c.args].join(' ')).join());
    assert.equal(new Set(cmds).size, IDS.length, `commandes confondues : ${cmds.join(' | ')}`);
  });

  // HAR-4 : --runs demande un entier d'au moins 1. Absent, nul ou non numérique, il donnait NaN ou 0 lancement, puis un TypeError
  // dans cell() : le banc refuse maintenant par une erreur d'usage (code 2, sur stderr), avant tout build. Le processus du banc
  // lancé ici ne construit ni ne lance aucune implémentation.
  test('HAR-4 : --runs demande un entier d’au moins 1', () => {
    for (const mauvais of [[], [''], ['0'], ['-1'], ['2.5'], ['1e3'], ['0x10'], [' 3'], ['abc'], ['go'], ['9007199254740993'], ['99999999999999999999']]) {
      assert.throws(() => parseArgs(['py', '--runs', ...mauvais]), /--runs demande un entier/, `--runs ${JSON.stringify(mauvais)} accepté`);
    }
    assert.deepEqual(parseArgs([]), { runs: 11, ids: [] });
    assert.deepEqual(parseArgs(['py', '--runs', '3', 'go']), { runs: 3, ids: ['py', 'go'] });
    assert.deepEqual(parseArgs(['--runs', '1']), { runs: 1, ids: [] });
    const banc = fileURLToPath(new URL('./bench.ts', import.meta.url));
    const r = spawnSync(process.execPath, [banc, 'py', '--runs', '0'], { encoding: 'utf8' });
    assert.ifError(r.error);
    assert.equal(r.status, 2, `code ${r.status}, stderr ${r.stderr.slice(0, 200)}`);
    assert.equal(r.stdout, '', 'une erreur d’usage n’écrit rien sur stdout');
    assert.match(r.stderr, /^bench: --runs demande un entier d’au moins 1 \(reçu : "0"\)\nusage : /);
  });

  // HAR-5 : l'A/B apparié du banc (`--contre REF`). L'ordre s'alterne (A puis B aux tours pairs, B puis A aux impairs) et la paire
  // se rend toujours [A, B]. Le résumé est la médiane des différences B − A, non la différence des médianes : la dérive de la machine
  // entre deux tours touche A et B ensemble, et la différence des médianes la rejoue. A part de la racine de la révision, B de ce
  // dépôt : sinon l'A/B comparerait l'arbre de travail à lui-même, sans écart. L'espion ne lance rien; seuls le nettoyage et l'A/B entier (`main`) écrivent sur le disque, chacun dans un dossier provisoire.
  test('HAR-5 : l’A/B apparié alterne l’ordre, rend la médiane des différences B − A, et lance A depuis la révision, B depuis le dépôt', () => {
    const ordre: string[] = [];
    for (let i = 0; i < 4; i++) pair(i, () => ordre.push(`A${i}`), () => ordre.push(`B${i}`));
    assert.deepEqual(ordre, ['A0', 'B0', 'B1', 'A1', 'A2', 'B2', 'B3', 'A3'], 'l’ordre ne s’alterne pas');
    assert.deepEqual(pair(1, () => 'a', () => 'b'), ['a', 'b'], 'pair ne rend pas [A, B] à un tour impair');

    // B gagne 4 paires sur 5 de 0,5 ms alors que sa médiane dépasse celle de A : seule la médiane des différences le voit.
    assert.deepEqual(paired([[10, 9.5], [50, 49.5], [12, 11.5], [48, 47.5], [30, 31]]), { a: 30, b: 31, d: -0.5, faster: 4 });
    // Une égalité n'est pas un gain : « B < A » est strict.
    assert.equal(paired([[5, 5], [7, 6]]).faster, 1, 'une paire à égalité compte comme un gain de B');
    // Un nombre pair de paires : toute médiane du banc (A, B et B − A) est la moyenne des deux éléments centraux, non l'élément haut
    // ni l'élément bas. A : 10 20 30 40, B : 12 26 38 52, B − A : 2 6 8 12.
    assert.deepEqual(paired([[10, 12], [20, 26], [30, 38], [40, 52]]), { a: 25, b: 32, d: 7, faster: 0 }, 'la médiane d’un nombre pair n’est pas la moyenne des deux éléments centraux');

    const depot = fileURLToPath(new URL('..', import.meta.url));
    const revision = join(tmpdir(), 'fib-har5-revision');
    for (const id of IDS) {
      const calls = spy(() => abRound(0, id, ['10'], revision));
      assert.equal(calls.length, 2, `${id} : un tour, deux lancements`);
      const [a, b] = calls;
      assert.deepEqual([a.file, ...a.args], [...command(id, revision), '--time', '10'], `${id} : A ne lance pas la commande de la révision`);
      assert.equal(a.options.cwd, revision, `${id} : A ne part pas de la racine de la révision`);
      assert.deepEqual([b.file, ...b.args], [...command(id), '--time', '10'], `${id} : B ne lance pas la commande du dépôt`);
      assert.equal(b.options.cwd, depot, `${id} : B ne part pas de la racine du dépôt`);
    }
    // Go et Rust tirent leur binaire de la racine donnée; TypeScript et Python s'y lancent comme dossier courant (le dossier ci-dessus).
    for (const id of ['go', 'rust']) assert.ok(command(id, revision)[0].startsWith(revision), `${id} : le binaire ne vient pas de la révision`);
    // present et build suivent la racine donnée : une racine sans dossier n'a aucune implémentation, et un build part de cette racine.
    assert.deepEqual(present(revision), [], 'present ne lit pas la racine donnée');
    for (const id of ['go', 'rust', 'ts']) {
      const compile = spy(() => build(id, revision));
      assert.equal(compile.length, 1, `${id} : un build, un lancement`);
      assert.equal(compile[0].options.cwd, revision, `${id} : le build ne part pas de la racine de la révision`);
    }
    // Aux tours impairs, B part le premier, et chacun garde sa racine.
    const [premier, second] = spy(() => abRound(1, 'go', ['10'], revision));
    assert.deepEqual([premier.options.cwd, second.options.cwd], [depot, revision], 'abRound n’alterne pas l’ordre des lancements');

    // Une mesure qui échoue dit de quel côté : le code non nul d'une CLI ne dit pas, seul, si la révision ou l'arbre de travail est en cause.
    // Aux deux parités du tour, puisque l'ordre des lancements s'inverse.
    for (const [cwd, cote] of [[revision, /^côté A \(la révision REF, /], [depot, /^côté B \(l’arbre de travail\) : /]] as const) {
      for (const tour of [0, 1]) {
        assert.throws(
          () => spy(() => abRound(tour, 'go', ['10'], revision), (c) => (c.options.cwd === cwd ? { status: 1, stderr: 'boum' } : {})),
          (e: Error) => cote.test(e.message) && e.message.endsWith('go 10 : code 1, boum'),
          `tour ${tour} : l’échec d’un lancement depuis ${cwd} ne nomme pas son côté`,
        );
      }
    }

    // La politique de mesure que le banc et l'A/B partagent (`rounds`) : le tour 0 chauffe les caches et s'écarte, les tours suivants se
    // gardent, et les scénarios s'entrelacent dans chaque tour pour que la dérive de la machine les touche tous.
    const vus: [i: number, args: string[]][] = [];
    const gardes = rounds(2, (i, args) => (vus.push([i, args]), i));
    const S = gardes.length;
    assert.ok(S > 1, 'rounds ne rend pas un échantillon par scénario');
    assert.deepEqual(gardes, Array.from({ length: S }, () => [1, 2]), 'le tour 0 est gardé, ou un tour manque');
    assert.equal(vus.length, 3 * S, 'rounds ne joue pas 1 tour de chauffe et 2 tours gardés pour chaque scénario');
    assert.equal(new Set(vus.slice(0, S).map(([, args]) => args.join(' '))).size, S, 'deux scénarios ont les mêmes arguments');
    vus.forEach(([i, args], k) => {
      assert.equal(i, Math.floor(k / S), `rendez-vous ${k} : les scénarios ne s’entrelacent pas par tour`);
      assert.deepEqual(args, vus[k % S][1], `rendez-vous ${k} : les scénarios ne se suivent pas dans le même ordre à chaque tour`);
    });

    // L'A/B ne joue qu'un nombre pair de paires, et abPairs, là où vit l'alternance, le refuse avant de lancer quoi que ce soit : sur un nombre
    // impair, un côté part le premier dans une paire de plus, et la médiane des différences retombe dans ce groupe. `main` arrondit au pair
    // supérieur avant de l'appeler (ci-dessous), mais un autre appelant (`versus`, un test) ne l'arrondirait pas. Nul, négatif ou décimal : de même.
    for (const mauvais of [1, 3, 101, 0, -2, 2.5, Number.NaN]) {
      const lances: Call[] = [];
      assert.throws(() => spy(() => abPairs('go', mauvais, revision), (c) => (lances.push(c), {})), /un nombre pair de paires, au moins 2 \(reçu : /, `abPairs accepte ${mauvais} paires`);
      assert.equal(lances.length, 0, `abPairs lance ${lances.length} mesures avant de refuser ${mauvais} paires`);
    }

    // L'A/B joue cette politique par abRound : chaque rendez-vous passe l'indice du tour et la racine de la révision. 3 tours (0, 1 et 2),
    // donc les deux parités; deux paires gardées par scénario. Un tour pair lance A (la révision) d'abord, un tour impair B d'abord.
    for (const id of IDS) {
      let paires: [number, number][][] = [];
      const lancements = spy(() => void (paires = abPairs(id, 2, revision)));
      assert.deepEqual(paires.map((p) => p.length), Array.from({ length: S }, () => 2), `${id} : l’A/B ne garde pas les paires des tours 1 et 2 seuls`);
      assert.equal(lancements.length, 2 * 3 * S, `${id} : l’A/B ne joue pas 3 tours de ${S} scénarios, deux lancements chacun`);
      for (let r = 0; r < 3; r++) {
        for (let s = 0; s < S; s++) {
          const [premier, second] = lancements.slice(2 * (r * S + s), 2 * (r * S + s) + 2);
          const [a, b] = r % 2 === 0 ? [premier, second] : [second, premier];
          const ou = `${id}, tour ${r}, scénario ${s}`;
          assert.deepEqual([a.file, ...a.args], [...command(id, revision), '--time', ...vus[s][1]], `${ou} : A ne lance pas la commande de la révision`);
          assert.equal(a.options.cwd, revision, `${ou} : A ne part pas de la racine de la révision`);
          assert.deepEqual([b.file, ...b.args], [...command(id), '--time', ...vus[s][1]], `${ou} : B ne lance pas la commande du dépôt`);
          assert.equal(b.options.cwd, depot, `${ou} : B ne part pas de la racine du dépôt`);
        }
      }
    }

    // --contre REF : la révision se lit après l'option, et une valeur absente ou qui ressemble à une option est une erreur d'usage.
    assert.deepEqual(parseArgs(['py', '--contre', 'HEAD', '--runs', '3']), { runs: 3, ids: ['py'], ref: 'HEAD' });
    for (const mauvais of [[], [''], ['-x'], ['--runs', '3']]) {
      assert.throws(() => parseArgs(['py', '--contre', ...mauvais]), /--contre demande une révision/, `--contre ${JSON.stringify(mauvais)} accepté`);
    }

    // Le défaut de --runs : 11 pour le banc, 100 sous --contre (une centaine de paires pour un écart de quelques pour cent, ADR-0004, et un
    // nombre pair : l'ordre alterne, donc un nombre impair de paires donne à un côté une paire de plus qui part le premier, et la médiane
    // retombe dans ce groupe), quel que soit l'ordre des options; un --runs explicite l'emporte, avant ou après --contre.
    assert.equal(parseArgs(['py']).runs, 11, 'le défaut du banc a changé');
    assert.equal(parseArgs(['py', '--contre', 'HEAD']).runs, 100, 'l’A/B n’a pas son défaut de 100 paires');
    assert.equal(parseArgs(['--contre', 'HEAD', 'py']).runs, 100, 'le défaut de l’A/B dépend de l’ordre des options');
    assert.equal(parseArgs(['py', '--runs', '5', '--contre', 'HEAD']).runs, 5, 'un --runs avant --contre est écrasé');
    assert.equal(parseArgs(['py', '--contre', 'HEAD', '--runs', '5']).runs, 5, 'un --runs après --contre est écrasé');

    // Un argument qui n'est ni un identifiant du PRD § 5 ni une option est une erreur d'usage, sans build : sinon `build()` lève une
    // pile pour un identifiant inconnu, et une option mal écrite (`--run`) passe pour un identifiant.
    for (const inconnu of ['perl', 'GO', '--run', '--contres', '']) {
      assert.throws(() => parseArgs(['py', inconnu]), /argument inconnu : /, `${JSON.stringify(inconnu)} accepté`);
      assert.throws(() => parseArgs([inconnu, '--contre', 'HEAD']), /argument inconnu : /, `${JSON.stringify(inconnu)} accepté sous --contre`);
    }

    // Le banc entier, `main`, sous l'espion : une erreur d'usage ou de révision s'écrit sur stderr en une ligne `bench: …`, sans pile, avec
    // son code (2 pour l'usage, 1 pour la révision), et avant tout build et toute extraction. Seuls `git rev-parse` et `git ls-tree`
    // se lancent, jamais `git read-tree` ni un compilateur : l'espion rendrait ces appels réussis, et ils laisseraient .build/revisions/<sha>.
    const banc = (argv: string[], reply?: (call: Call) => Partial<Reply>, revisions?: string) => {
      const sortie: string[] = [];
      const err: string[] = [];
      const real = { log: console.log, error: console.error };
      let code: number | undefined;
      let calls: Call[] = [];
      Object.assign(console, { log: (...a: unknown[]) => sortie.push(a.join(' ')), error: (...a: unknown[]) => err.push(a.join(' ')) });
      try {
        calls = spy(() => void (code = main(argv, revisions)), reply);
      } finally {
        Object.assign(console, real);
      }
      return { code, calls, stdout: sortie.join('\n'), stderr: err.join('\n') };
    };
    const sansPile = (stderr: string) => assert.doesNotMatch(stderr, /\n\s+at /, 'une pile accompagne l’erreur');
    const sha = 'a'.repeat(40);
    const arbre = (...dossiers: string[]) => (c: Call): Partial<Reply> =>
      c.args[0] === 'rev-parse' ? { stdout: `${sha}\n` } : c.args[0] === 'ls-tree' ? { stdout: dossiers.map((d) => `${d}\0`).join('') } : {};
    const seulementLecture = (calls: Call[]) =>
      assert.ok(calls.every((c) => c.file === 'git' && ['rev-parse', 'ls-tree'].includes(c.args[0])), `un appel autre qu’une lecture de révision : ${calls.map((c) => [c.file, ...c.args].join(' ')).join(' ; ')}`);

    const inconnu = banc(['perl']);
    assert.deepEqual([inconnu.code, inconnu.calls.length, inconnu.stdout], [2, 0, ''], 'un identifiant inconnu doit refuser (code 2) sans rien lancer');
    assert.match(inconnu.stderr, /^bench: argument inconnu : "perl"[^\n]*\nusage : /);
    sansPile(inconnu.stderr);

    const introuvable = banc(['go', '--contre', 'nope'], (c) => (c.args[0] === 'rev-parse' ? { status: 1 } : {}));
    assert.deepEqual([introuvable.code, introuvable.stdout], [1, ''], 'une révision introuvable doit refuser (code 1)');
    assert.equal(introuvable.stderr, 'bench: révision introuvable : nope');
    seulementLecture(introuvable.calls);

    // Chaque langage demandé doit être un dossier de REF, avant la moindre extraction : ici REF n'a ni rust ni python.
    const absent = banc(['go', 'rust', 'py', '--contre', 'HEAD'], arbre('go', 'ts'));
    assert.deepEqual([absent.code, absent.stdout], [1, ''], 'un langage absent de REF doit refuser (code 1)');
    assert.equal(absent.stderr, 'bench: rust, py : dossier absent de HEAD');
    seulementLecture(absent.calls);
    assert.ok(absent.calls.some((c) => c.args.includes(sha)), 'le dossier se lit dans la révision résolue, non dans le nom donné');
    // Python se cherche sous son dossier, `python`, non sous son identifiant `py` : ici seul go manque.
    const dossier = banc(['py', 'go', '--contre', 'HEAD'], arbre('rust', 'ts', 'python'));
    assert.deepEqual([dossier.code, dossier.stderr], [1, 'bench: go : dossier absent de HEAD'], 'un langage se cherche sous un autre nom que son dossier');
    seulementLecture(dossier.calls);
    // Sans identifiant, ce sont les langages présents dans ce dépôt (les quatre ici) qui doivent se trouver dans REF.
    const defaut = banc(['--contre', 'HEAD'], arbre('go', 'rust', 'ts'));
    assert.deepEqual([defaut.code, defaut.stderr], [1, 'bench: py : dossier absent de HEAD'], 'les langages par défaut ne sont pas cherchés dans REF');
    seulementLecture(defaut.calls);

    // Le banc ordinaire, `main` jusqu'au tableau, sous l'espion : `rounds(runs, …)` y joue 1 tour de chauffe et `runs` tours gardés, tous depuis
    // le dépôt. Sans `--contre`, ni lecture ni extraction de révision. Un `rounds(runs - 1, …)` de `table` ne lance que le tour de chauffe et
    // rend des médianes vides (NaN), que seul le compte des lancements voit.
    const ordinaire = banc(['go', '--runs', '1']);
    assert.deepEqual([ordinaire.code, ordinaire.stderr], [0, ''], 'le banc ordinaire doit réussir sans rien écrire sur stderr');
    assert.ok(ordinaire.calls.every((c) => c.file !== 'git'), 'le banc ordinaire lit ou extrait une révision');
    assert.deepEqual(ordinaire.calls.filter((c) => c.file === 'go').map((c) => c.options.cwd), [depot], 'le banc ordinaire ne compile pas Go une fois, depuis le dépôt');
    const mesuresBanc = ordinaire.calls.filter((c) => c.file !== 'go');
    assert.equal(mesuresBanc.length, 2 * S, 'le banc ne joue pas 1 tour de chauffe et 1 tour gardé de chaque scénario');
    for (const c of mesuresBanc) assert.deepEqual([c.file, c.options.cwd], [command('go')[0], depot], 'le banc ordinaire lance autre chose que le binaire du dépôt');
    const lignesBanc = ordinaire.stdout.split('\n');
    assert.match(lignesBanc[0], /^médianes sur 1 exécutions /, 'l’en-tête du banc ordinaire a changé');
    assert.equal(lignesBanc.length, 2 + S, 'le banc ordinaire n’écrit pas son en-tête, ses colonnes et une ligne par scénario');
    assert.doesNotMatch(ordinaire.stdout, /NaN/, 'le banc ordinaire écrit une médiane vide');

    // L'A/B entier, `main` sous l'espion jusqu'au tableau, avec une révision déjà extraite : `extract` réutilise <revisions>/<sha> sans lancer
    // git, donc seuls `git rev-parse`, `git ls-tree`, deux builds et les lancements de mesure passent par l'espion. `abPairs` ne garde pas, seul, la
    // racine que `versus` lui passe : un `abPairs(id, runs, ROOT)` dans `versus` laisserait HAR-5 vert, et l'A/B comparerait le dépôt à lui-même.
    // Un `build(id, rev)` retiré ou remplacé par `build(id)` y survivrait aussi, et ne se verrait qu'à une extraction neuve.
    // L'ordre alterne (A part le premier aux tours pairs), donc seul un nombre pair de paires le répartit à égalité : sur un nombre impair,
    // un côté part le premier dans une paire de plus, et la médiane des différences retombe dans ce groupe avec son effet d'ordre. Un --runs
    // impair est arrondi au pair supérieur, et le banc le dit sur stderr (`--runs 1` : 2 paires, `--runs 3` : 4; `--runs 2` : 2, sans mot).
    const base = mkdtempSync(join(tmpdir(), 'fib-har5-ab-'));
    try {
      const rev = join(base, sha);
      mkdirSync(rev);
      for (const [demande, paires] of [[1, 2], [2, 2], [3, 4]]) {
        const ou = `--runs ${demande}`;
        const ab = banc(['go', '--runs', String(demande), '--contre', 'HEAD'], arbre('go', 'rust', 'ts', 'python'), base);
        const dit = demande === paires ? '' : `bench: --runs ${demande} est impair sous --contre, arrondi à ${paires} paires : A et B doivent partir le premier à égalité`;
        assert.deepEqual([ab.code, ab.stderr], [0, dit], `${ou} : l’A/B doit réussir, et ne dire l’arrondi que pour un nombre impair`);
        assert.match(ab.stdout, new RegExp(`^A/B apparié sur ${paires} paires [^\\n]*A = HEAD \\(aaaaaaa\\), B = `), `${ou} : l’en-tête ne nomme pas ${paires} paires, REF et son SHA court`);
        // Aucun appel git hors la lecture de la révision : la révision déjà extraite est réutilisée.
        seulementLecture(ab.calls.filter((c) => c.file === 'git'));
        // Un build part de la racine de la révision, l'autre du dépôt, dans un ordre quelconque.
        assert.deepEqual(ab.calls.filter((c) => c.file === 'go').map((c) => c.options.cwd).sort(), [depot, rev].sort(), `${ou} : les deux builds ne partent pas du dépôt et de la révision`);
        // 1 tour de chauffe et `paires` tours gardés : (paires + 1) × S rendez-vous, deux lancements chacun; la moitié, A, part de la racine de la révision.
        const mesures = ab.calls.filter((c) => c.file !== 'git' && c.file !== 'go');
        assert.equal(mesures.length, 2 * (paires + 1) * S, `${ou} : l’A/B ne joue pas 1 tour de chauffe et ${paires} tours gardés de chaque scénario, deux lancements chacun`);
        const de = (racine: string) => mesures.filter((c) => c.options.cwd === racine && c.file === command('go', racine)[0]);
        assert.equal(de(rev).length, (paires + 1) * S, `${ou} : la moitié des lancements (A) ne part pas de la racine de la révision, avec son binaire`);
        assert.equal(de(depot).length, (paires + 1) * S, `${ou} : la moitié des lancements (B) ne part pas du dépôt, avec son binaire`);
        // Dans les tours gardés (1 à paires), chaque côté part le premier dans la moitié des paires de chaque scénario : le rendez-vous r × S + s
        // est la paire de lancements 2k et 2k + 1, et le tour 0, écarté, ne compte pas.
        for (let s = 0; s < S; s++) {
          let aDabord = 0;
          for (let r = 1; r <= paires; r++) if (mesures[2 * (r * S + s)].options.cwd === rev) aDabord++;
          assert.equal(aDabord, paires / 2, `${ou}, scénario ${s} : A ne part pas le premier dans la moitié des paires`);
        }
        const lignes = ab.stdout.split('\n');
        assert.equal(lignes.length, 2 + S, `${ou} : l’A/B n’écrit pas son en-tête, ses colonnes et une ligne par scénario`);
        for (const ligne of lignes.slice(2)) assert.match(ligne, new RegExp(`/${paires}$`), `${ou} : une ligne du tableau ne compte pas ${paires} paires : ${ligne}`);
      }
    } finally {
      fs.rmSync(base, { recursive: true, force: true });
    }

    // Nettoyer .build/revisions/ sans suivre les liens : chaque révision porte un lien (jonction sous Windows) vers le node_modules de ce
    // dépôt, que `Remove-Item -Recurse` de Windows PowerShell 5.1 peut traverser pour vider sa cible. Le banc détache d'abord le lien,
    // puis supprime le dossier, et ne touche ni la cible ni le reste de .build/.
    const tmp = mkdtempSync(join(tmpdir(), 'fib-har5-nettoyage-'));
    try {
      const racine = join(tmp, '.build');
      const revisions = join(racine, 'revisions');
      const rev = join(revisions, sha);
      const cible = join(tmp, 'node_modules');
      const lien = join(rev, 'node_modules');
      mkdirSync(rev, { recursive: true });
      mkdirSync(cible);
      writeFileSync(join(cible, 'precieux.txt'), 'x');
      writeFileSync(join(rev, 'package.json'), '{}');
      writeFileSync(join(racine, 'fib'), 'binaire du dépôt');
      writeFileSync(join(revisions, `${'b'.repeat(40)}.index`), ''); // le reste d'une extraction interrompue
      symlinkSync(cible, lien, 'junction');
      assert.ok(lstatSync(lien).isSymbolicLink(), 'le lien de la sonde n’en est pas un');

      const real = fs.rmSync;
      let lienAuRm: boolean | undefined;
      Object.assign(fs, { rmSync: (...a: Parameters<typeof real>) => ((lienAuRm ??= lstatSync(lien, { throwIfNoEntry: false }) !== undefined), real(...a)) });
      syncBuiltinESMExports();
      let nettoye: ReturnType<typeof banc>;
      try {
        nettoye = banc(['--nettoyer'], undefined, revisions);
      } finally {
        Object.assign(fs, { rmSync: real });
        syncBuiltinESMExports();
      }
      assert.deepEqual([nettoye.code, nettoye.calls.length, nettoye.stderr], [0, 0, ''], '--nettoyer doit réussir sans rien lancer');
      assert.equal(lienAuRm, false, 'le lien est encore là quand rmSync supprime le dossier : il pourrait en traverser la cible');
      assert.ok(!existsSync(revisions), '--nettoyer ne vide pas .build/revisions/');
      assert.ok(existsSync(join(cible, 'precieux.txt')), '--nettoyer a vidé la cible du lien');
      assert.ok(existsSync(join(racine, 'fib')), '--nettoyer a touché autre chose que .build/revisions/');
      assert.equal(banc(['--nettoyer'], undefined, revisions).code, 0, 'rien à nettoyer doit réussir aussi');
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
    // --nettoyer s'emploie seul : combiné, il supprimerait sans que l'appelant l'ait demandé, ou ignorerait le reste.
    assert.deepEqual(parseArgs(['--nettoyer']), { runs: 11, ids: [], clean: true });
    for (const avec of [['py'], ['--runs', '3'], ['--contre', 'HEAD']]) {
      assert.throws(() => parseArgs(['--nettoyer', ...avec]), /--nettoyer s’emploie seul/, `--nettoyer ${avec.join(' ')} accepté`);
      assert.throws(() => parseArgs([...avec, '--nettoyer']), /--nettoyer s’emploie seul/, `${avec.join(' ')} --nettoyer accepté`);
    }
  });

  // HAR-6 : ADR-0001. La boucle différentielle (`contracts/differentiel.ts`) est un outil de recherche. Si le contrat la lançait ou
  // l'importait, un écart entre implémentations deviendrait un verdict du contrat, sans attendu pour le trancher : l'oracle par vote
  // que l'ADR-0001 écarte. Aucun script `check*` ni `contract` ne l'atteint, en suivant de proche en proche les scripts qu'ils lancent
  // (`npm run`, `node --run`, `npm test` et ses pairs, avec les crochets `pre` et `post` que npm joue) puis les fichiers que ces
  // commandes lancent, globs développés (le `node --test "ts/**/*.test.ts"` de check:ts), et le test de contrat ne l'importe pas, même
  // par un module qu'il importe. Le prototype d'A-2 laissait passer un `contract` qui lançait `npm run recherche`, un script qui lançait
  // la boucle : 117 sur 117. Le contrôle ne lance rien : il lit package.json et les sources, et ses témoins des globs et des imports
  // n'écrivent que des fichiers d'essai sous tmpdir(), qu'ils retirent.
  const LOOP = /differentiel/;
  // Les scripts qu'une commande lance. `npm run <script>`, `npm run-script <script>` et leurs alias `rum` et `urn`, avec des options avant
  // ou après et le nom entre guillemets ou non; `node --run <script>` (ou `--run=<script>`); et les scripts que npm nomme lui-même :
  // `npm test` (`tst`, `t`), `npm start`, `npm stop` et `npm restart`, qui joue `restart` ou, à défaut, `stop` puis `start`.
  const NPM_RUN = /\bnpm\s+(?:-\S+\s+)*(?:run-script|run|rum|urn)\s+(?:-\S+\s+)*["']?([^\s;&|<>()"']+)/g;
  const NODE_RUN = /\bnode\s+(?:-\S+\s+)*--run(?:=|\s+)["']?([^\s;&|<>()"']+)/g;
  const NPM_NAMED = /\bnpm\s+(?:-\S+\s+)*(test|tst|t|start|stop|restart)(?![\w:.-])/g;
  const NAMED: Record<string, string[]> = { test: ['test'], tst: ['test'], t: ['test'], start: ['start'], stop: ['stop'], restart: ['restart', 'stop', 'start'] };
  const launchedScripts = (command: string): string[] => [
    ...[...command.matchAll(NPM_RUN)].map((m) => m[1]),
    ...[...command.matchAll(NODE_RUN)].map((m) => m[1]),
    ...[...command.matchAll(NPM_NAMED)].flatMap((m) => NAMED[m[1]]),
  ];
  // Les commandes des scripts atteints depuis `roots` : les leurs, et de proche en proche celles des scripts qu'elles lancent.
  const commandsReached = (scripts: Record<string, string>, roots: readonly string[]): string[] => {
    const seen = new Set<string>();
    const visit = (name: string): void => {
      if (seen.has(name) || !Object.hasOwn(scripts, name)) return;
      seen.add(name);
      visit(`pre${name}`);
      visit(`post${name}`);
      launchedScripts(scripts[name]).forEach(visit);
    };
    roots.forEach(visit);
    return [...seen].map((name) => scripts[name]);
  };
  const isFile = (f: string) => existsSync(f) && statSync(f).isFile();
  // Les fichiers sous `dir`, sans les dossiers que le parcours d'un glob n'a pas à lire : dépendances, historique, sorties de build.
  const SKIPPED = new Set(['node_modules', '.git', '.build', 'target']);
  const filesUnder = (dir: string): string[] =>
    existsSync(dir)
      ? readdirSync(dir, { withFileTypes: true }).flatMap((e) => (SKIPPED.has(e.name) ? [] : e.isDirectory() ? filesUnder(join(dir, e.name)) : [join(dir, e.name)]))
      : [];
  // Les fichiers qu'un jeton de commande nomme sous `root` : lui-même s'il n'a ni `*` ni `?`, sinon ceux que le glob désigne. `**` vaut zéro
  // dossier ou plus, `*` et `?` ne franchissent pas un `/`. Ni `[…]` ni `{…}` : ce sont des caractères comme les autres (lacune du catalogue).
  const expandGlob = (token: string, root: string): string[] => {
    const parts = token.split('/');
    const at = parts.findIndex((part) => /[*?]/.test(part));
    if (at === -1) return [join(root, token)];
    const base = join(root, ...parts.slice(0, at));
    const source = parts.slice(at).join('/').replace(/\*\*\/|\*\*|\*|\?|[.+^${}()|[\]\\]/g, (m) => (m === '**/' ? '(?:.*/)?' : m === '**' ? '.*' : m === '*' ? '[^/]*' : m === '?' ? '[^/]' : `\\${m}`));
    const pattern = new RegExp(`^${source}$`);
    return filesUnder(base).filter((f) => pattern.test(relative(base, f).split(sep).join('/')));
  };
  // Les fichiers `.ts` et `.js` que ces commandes lancent : chaque jeton d'une commande, guillemets retirés, glob développé.
  const launchedFiles = (commands: readonly string[], root: string): string[] =>
    commands
      .flatMap((c) => c.split(/\s+/))
      .map((t) => t.replace(/^["']|["']$/g, ''))
      .filter(Boolean)
      .flatMap((t) => expandGlob(t, root))
      .filter((f) => /\.[cm]?[jt]s$/.test(f) && isFile(f));
  // Les chemins relatifs que `source` nomme. Ceux d'un `from`, d'un `import` (nu ou `import()`) et d'un `import.meta.resolve()` demandent
  // `./` ou `../` : sans eux, c'est un paquet. Celui d'un `new URL(…)` (comme HAR-4 vise le banc) se lit relativement à son second
  // argument, préfixe ou non, tant qu'il n'est ni un schéma (`node:`, `file:`) ni un chemin absolu. Entre ', " ou accents graves.
  const IMPORT = /(?:\bfrom|\bimport|\bimport\.meta\.resolve)\s*\(?\s*['"`](\.{1,2}\/[^'"`]+)['"`]/g;
  const NEW_URL = /\bnew\s+URL\s*\(\s*['"`](?!\/)([^'"`:]+)['"`]/g;
  const relatives = (source: string): string[] => [...source.matchAll(IMPORT), ...source.matchAll(NEW_URL)].map((m) => m[1]);
  // Les modules que `entry` importe, de proche en proche, lui compris. Seuls comptent les chemins qui mènent à un fichier du dépôt.
  const importClosure = (entry: string): string[] => {
    const seen = new Set<string>();
    const pending = [entry];
    for (let file = pending.pop(); file !== undefined; file = pending.pop()) {
      if (seen.has(file)) continue;
      seen.add(file);
      if (isFile(file)) for (const rel of relatives(readFileSync(file, 'utf8'))) pending.push(join(dirname(file), rel));
    }
    return [...seen];
  };

  test('HAR-6 : la boucle différentielle reste hors de check*, de contract et du test de contrat', () => {
    // Témoins du parcours des scripts : il suit `npm run` de proche en proche, avec ses options et ses crochets, et ne déborde pas.
    const exemple = { precontract: 'echo', contract: 'node a.ts && npm run "b"', b: 'npm --silent run-script c -- x', c: 'node contracts/differentiel.ts', d: 'echo' };
    assert.ok(commandsReached(exemple, ['contract']).some((c) => LOOP.test(c)), 'témoin : le parcours ne suit pas npm run de proche en proche');
    assert.ok(commandsReached({ ...exemple, b: 'echo', precontract: exemple.c }, ['contract']).some((c) => LOOP.test(c)), 'témoin : le parcours oublie le crochet pre');
    assert.ok(!commandsReached(exemple, ['d']).some((c) => LOOP.test(c)), 'témoin : le parcours atteint un script qu’aucun npm run ne lance');
    // Chaque façon de lancer un script, avec les crochets que npm joue autour : `cible` est le script que la commande lance, et il lance la boucle.
    const BOUCLE = 'node contracts/differentiel.ts';
    const lancements: [forme: string, commande: string, cible: string][] = [
      ['npm run', 'npm run cible', 'cible'],
      ['npm run, nom entre guillemets', 'npm run "cible"', 'cible'],
      ['npm rum', 'npm rum cible', 'cible'],
      ['npm urn', 'npm urn cible', 'cible'],
      ['node --run', 'node --run cible', 'cible'],
      ['node --run=', 'node --run=cible', 'cible'],
      ['node --run, option avant', 'node --no-warnings --run cible', 'cible'],
      ['npm test', 'npm test', 'test'],
      ['npm test --', 'npm --silent test -- x', 'test'],
      ['npm tst', 'npm tst', 'test'],
      ['npm t', 'npm t', 'test'],
      ['npm start', 'npm start', 'start'],
      ['npm stop', 'npm stop', 'stop'],
      ['npm restart', 'npm restart', 'restart'],
      ['npm restart, repli sur start', 'npm restart', 'start'],
      ['npm restart, repli sur stop', 'npm restart', 'stop'],
    ];
    for (const [forme, commande, cible] of lancements) {
      const atteint = (scripts: Record<string, string>) => commandsReached({ contract: `echo && ${commande}`, ...scripts }, ['contract']).some((c) => LOOP.test(c));
      assert.ok(atteint({ [cible]: BOUCLE }), `témoin : le parcours ne suit pas « ${forme} »`);
      for (const crochet of [`pre${cible}`, `post${cible}`]) {
        assert.ok(atteint({ [cible]: 'echo', [crochet]: BOUCLE }), `témoin : le parcours oublie le crochet ${crochet} de « ${forme} »`);
      }
      assert.ok(!atteint({ [`${cible}x`]: BOUCLE, [`x${cible}`]: BOUCLE }), `témoin : « ${forme} » atteint un script au nom voisin`);
    }
    assert.ok(!commandsReached({ contract: 'npm token list', test: BOUCLE }, ['contract']).some((c) => LOOP.test(c)), 'témoin : npm token atteint le script test (t pour test)');
    assert.ok(!commandsReached({ contract: 'npm run test:unit', test: BOUCLE }, ['contract']).some((c) => LOOP.test(c)), 'témoin : npm run test:unit atteint le script test');
    assert.ok(!commandsReached({ contract: 'node contracts/cli.test.ts', test: BOUCLE }, ['contract']).some((c) => LOOP.test(c)), 'témoin : le parcours atteint test sans qu’aucune commande le lance');

    // Témoins du parcours des imports, sur des fichiers écrits sous tmpdir() : la fermeture est transitive (a → b → c : un seul niveau
    // laissait passer `bench.ts` qui importe la boucle), et chaque forme d'un chemin relatif y est vue (une forme ignorée laissait passer
    // la boucle lancée par un `new URL(…, import.meta.url)` donné à `spawnSync`, comme HAR-4 lance le banc, ou importée par un `import()`
    // entre accents graves). Un chemin qui nomme un dossier (le `new URL('../..', …)` de ts/src/cli.test.ts) ne se lit pas, et un import nu
    // (`node:fs`) ne se suit pas. Les sources de ces témoins sont des chaînes : le motif les lit aussi ici, sans mener à un fichier de la boucle.
    const dossier = mkdtempSync(join(tmpdir(), 'fib-har6-'));
    try {
      const ecrire = (nom: string, source = 'export {};\n') => writeFileSync(join(dossier, nom), source);
      const fermeture = (nom: string) => importClosure(join(dossier, nom)).map((f) => relative(dossier, f)).sort();
      ecrire('a.ts', "import { b } from './b.ts';\n");
      ecrire('b.ts', 'import "./c.ts";\n');
      ecrire('c.ts');
      assert.deepEqual(fermeture('a.ts'), ['a.ts', 'b.ts', 'c.ts'], 'témoin : la fermeture des imports n’est pas transitive');
      ecrire('t.ts');
      const formes: Record<string, string> = {
        from: "import { t } from './t.ts';",
        'import nu': 'import "./t.ts";',
        'import()': "export const t = () => import('./t.ts');",
        'import() entre accents graves': 'export const t = () => import(`./t.ts`);',
        'new URL': "export const t = new URL('./t.ts', import.meta.url);",
        'new URL sans préfixe': "export const t = new URL('t.ts', import.meta.url);",
        'new URL entre guillemets doubles et espaces': 'export const t = new  URL( "t.ts" , import.meta.url);',
        'import.meta.resolve': "export const t = import.meta.resolve('./t.ts');",
        'import.meta.resolve entre guillemets doubles': 'export const t = import.meta.resolve( "./t.ts" );',
      };
      for (const [forme, source] of Object.entries(formes)) {
        ecrire('f.ts', `${source}\n`);
        assert.deepEqual(fermeture('f.ts'), ['f.ts', 't.ts'], `témoin : le parcours des imports ignore la forme « ${forme} »`);
      }
      mkdirSync(join(dossier, 'd'));
      ecrire('f.ts', "import { x } from 'node:fs';\nexport const d = new URL('./d', import.meta.url);\n");
      assert.deepEqual(fermeture('f.ts'), ['d', 'f.ts'], 'témoin : un import nu se suit, ou un chemin qui nomme un dossier se lit');
    } finally {
      rmSync(dossier, { recursive: true, force: true });
    }

    // Témoins du développement des globs, sur des fichiers écrits sous tmpdir() : `node --test "ts/**/*.test.ts"` (check:ts) lance des fichiers
    // qu'aucun jeton ne nomme, et un jeton dont le fichier n'existe pas était filtré. `**` vaut zéro dossier ou plus, `*` et `?` s'arrêtent
    // au `/`, les points sont des points, les dossiers de dépendances ne se lisent pas, et un jeton sans joker se rend tel quel.
    const arbre = mkdtempSync(join(tmpdir(), 'fib-har6-glob-'));
    try {
      for (const dir of ['d/sub/deep', 'd/node_modules', 'e']) mkdirSync(join(arbre, dir), { recursive: true });
      const fichiers = ['d/a.test.ts', 'd/x.ts', 'd/x.ts.txt', 'd/xtestxts', 'd/sub/b.test.ts', 'd/sub/deep/c.test.ts', 'd/node_modules/n.test.ts', 'e/f.js', 'f.js'];
      for (const f of fichiers) writeFileSync(join(arbre, f), 'export {};\n');
      const developpe = (token: string) => expandGlob(token, arbre).map((f) => relative(arbre, f).split(sep).join('/')).sort();
      assert.deepEqual(developpe('d/**/*.test.ts'), ['d/a.test.ts', 'd/sub/b.test.ts', 'd/sub/deep/c.test.ts'], 'témoin : ** ne vaut pas zéro dossier ou plus, ou un point est un joker, ou node_modules se lit');
      assert.deepEqual(developpe('d/*.test.ts'), ['d/a.test.ts'], 'témoin : * franchit un /');
      assert.deepEqual(developpe('d/sub/?.test.ts'), ['d/sub/b.test.ts'], 'témoin : ? ne vaut pas un caractère');
      assert.deepEqual(developpe('d/sub/??.test.ts'), [], 'témoin : ? vaut zéro ou plusieurs caractères');
      assert.deepEqual(developpe('d?sub/b.test.ts'), [], 'témoin : ? franchit un /');
      assert.deepEqual(developpe('d/sub/**'), ['d/sub/b.test.ts', 'd/sub/deep/c.test.ts'], 'témoin : ** final ne désigne pas tout le dossier');
      assert.deepEqual(developpe('**/f.js'), ['e/f.js', 'f.js'], 'témoin : ** en tête ne vaut pas zéro dossier ou plus');
      assert.deepEqual(developpe('d/x.ts'), ['d/x.ts'], 'témoin : un jeton sans joker n’est pas rendu tel quel');
      assert.deepEqual(developpe('absent/**/*.ts'), [], 'témoin : un glob sous un dossier absent rend des fichiers');
      // Les fichiers que des commandes lancent : un jeton entre guillemets, un glob développé, un jeton qui n'est pas un script.
      const lances = launchedFiles(['tsc --noEmit && node --test "d/**/*.test.ts"', 'node e/f.js absent.ts', 'echo d/x.ts.txt d/*.json'], arbre).map((f) => relative(arbre, f).split(sep).join('/')).sort();
      assert.deepEqual(lances, ['d/a.test.ts', 'd/sub/b.test.ts', 'd/sub/deep/c.test.ts', 'e/f.js'], 'témoin : les fichiers que des commandes lancent, globs compris');
    } finally {
      rmSync(arbre, { recursive: true, force: true });
    }

    const { scripts } = JSON.parse(readFileSync(join(ROOT, 'package.json'), 'utf8')) as { scripts: Record<string, string> };
    const roots = Object.keys(scripts).filter((name) => name === 'contract' || name.startsWith('check'));
    const commands = commandsReached(scripts, roots);
    assert.ok(commandsReached(scripts, ['check']).some((c) => c.includes('contracts/cli.test.ts')), 'témoin : le parcours de package.json ne voit pas le test de contrat');
    assert.deepEqual(commands.filter((c) => LOOP.test(c)), [], 'un script check* ou contract atteint la boucle');

    // Le test de contrat, et chaque fichier que ces commandes lancent (globs développés), avec leurs imports.
    const launched = launchedFiles(commands, ROOT);
    const reached = [...new Set([fileURLToPath(import.meta.url), ...launched].flatMap(importClosure))];
    // Les fichiers que `check:ts` lance par un glob, puis ce qu'ils importent : `ts/src/cli.ts` n'est lancé par aucune commande, mais `ts/src/cli.test.ts` l'importe.
    for (const f of [join('ts', 'src', 'cli.test.ts'), join('ts', 'src', 'fib.test.ts')]) assert.ok(launched.includes(join(ROOT, f)), `témoin : le glob de check:ts ne désigne pas ${f}`);
    for (const f of [join('ts', 'src', 'cli.ts'), join('ts', 'src', 'fib.ts'), join('contracts', 'oracle.ts')]) assert.ok(reached.includes(join(ROOT, f)), `témoin : le parcours des imports ne voit pas ${f}`);
    assert.ok(reached.includes(join(ROOT, 'contracts', 'lancement.ts')), 'témoin : le parcours des imports ne voit pas lancement.ts');
    assert.deepEqual(reached.map((f) => relative(ROOT, f)).filter((f) => LOOP.test(f)), [], 'un fichier que le contrat lance ou importe atteint la boucle par un import');
  });

  // HAR-7 : le contenu de la table de lancement (`contracts/lancement.ts`). HAR-1 et HAR-5 comparent le banc à `command(id)`, tiré de cette même
  // table, et HAR-3 n'exige que des commandes distinctes : aucune garde ne fixait ce que la table lance. Le retour de ts au `.ts` non compilé
  // (`ts/src/cli.ts`) et le retrait de `-I -S` pour Python passaient donc le contrat, alors que le banc aurait mesuré le retrait des types
  // de Node, ou `site` et PYTHON*, au lieu du programme (ADR-0004). Go par `go run` et Rust par `cargo run` mesureraient la chaîne d'outils.
  // Le contrôle ne lance rien et n'espionne rien : il compare `command(id)` à l'attendu, écrit ici sans la table, pour chaque identifiant de
  // `IDS` (les quatre du PRD § 5), un test chacun. Les attendus sont ceux du PRD § 5 : les binaires de Go et de Rust (suffixe `.exe` sous Windows,
  // comme dans la table) vus depuis la racine de ce dépôt, le JavaScript que tsc émet sous `.build/ts/`, et Python isolé (`-I`) sans `site` (`-S`).
  // La boucle court sur `IDS`, non sur ces attendus : un identifiant ajouté à la table sans attendu ici échoue sous son propre nom.
  const exe = process.platform === 'win32' ? '.exe' : '';
  const LANCEMENT: Record<string, { quoi: string; argv: string[] }> = {
    go: { quoi: 'le binaire de go build', argv: [join(ROOT, '.build', `fib${exe}`)] },
    rust: { quoi: 'le binaire de cargo build --release', argv: [join(ROOT, 'rust', 'target', 'release', `fib${exe}`)] },
    ts: { quoi: 'le JavaScript compilé par tsc, non la source', argv: [process.execPath, '.build/ts/cli.js'] },
    py: { quoi: 'python/fib.py sous -I -S', argv: ['python', '-I', '-S', 'python/fib.py'] },
  };
  for (const id of IDS) {
    const attendu: { quoi: string; argv: string[] } | undefined = LANCEMENT[id];
    test(`HAR-7 : ${id} se lance par ${attendu?.quoi ?? 'une commande sans attendu'}`, () => {
      assert.ok(attendu, `${id} : HAR-7 n’a pas d’attendu pour cet identifiant de lancement.ts`);
      assert.deepEqual(command(id), attendu.argv, `${id} : la table de lancement ne lance pas ${attendu.quoi}`);
    });
  }
});

const requested = process.argv.slice(2);
const here = present();
const ids = requested.length ? requested : here;

for (const id of ids) {
  describe(id, () => {
    if (!here.includes(id)) {
      test(`${id} : l’implémentation existe`, () => assert.fail(`implémentation absente : ${id}`));
      return;
    }
    before(() => build(id));
    for (const [cid, title, args, want, env] of CASES) test(`${cid} : ${title}`, () => verify(launch(id, args, { env }), want));
    test('CLI-V02 : forcer la couleur ne produit aucun octet ESC', () => V02(id));
    test('CLI-V04 : un stderr fermé ne change ni stdout ni le code', { skip: noShell }, () => V04(id));
  });
}
