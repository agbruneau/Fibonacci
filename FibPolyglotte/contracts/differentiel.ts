// Boucle différentielle : un outil de recherche (étape 9, `diagnosing-bugs`), hors de `npm run check` et de `npm run contract`
// (ADR-0001, HAR-6). Elle lance les implémentations présentes sur les mêmes entrées, puis regroupe celles dont les sorties diffèrent.
// Elle ne dit jamais laquelle a raison, ne lit pas l'oracle et ne fournit aucun attendu : un écart (CONTEXT.md) se tranche par le
// PRD § 4, puis par un cas du catalogue. Elle compare les implémentations d'une même révision, jamais deux révisions d'une même.
// Usage : node contracts/differentiel.ts [go|rust|ts|py …] [--dimension grammaire|valeurs|environnement|descripteurs]…
//         [--jetons K] [--revision REF]
// Code 0 sans écart, 1 avec au moins un écart, 2 pour une erreur d'usage, 3 si le témoin rougit, si la sélection ne produit aucune
// entrée ou si une extraction, un build ou un lancement échoue : le code 1 est le seul verdict.
// Hors contrat, donc hors de la boucle (PRD § 4) : un stdout fermé ou dont l'écriture échoue, un argument qui n'est pas de l'Unicode
// valide, les limites de ressources, les flux fusionnés. Leurs écarts sont permis, et aucune entrée ne les provoque.
import assert from 'node:assert/strict';
import { basename } from 'node:path';
import { build, decimalInt, extract, IDS, launch, present, resolve, REVISIONS, revisionArg, shown, type LaunchOptions } from './lancement.ts';

type Run = ReturnType<typeof launch>;

// Une entrée : les arguments de la CLI, et l'état dans lequel la lancer.
type Entry = { args: string[]; env?: LaunchOptions['env']; stderrClosed?: boolean };

// Ce que F10 interdit à stdout et à stderr, tel que le contrat l'applique : F10 se lit par X-4 (catalogue, ligne de traçabilité de F10), qui
// n'admet que l'ASCII imprimable (0x20 à 0x7E) et `\n`, que le découpage par ligne a déjà retiré. Tout autre caractère enfreint F10 : un octet
// de plus de 127, ESC (une séquence ANSI), le retour chariot (des fins de ligne `\n` partout), mais aussi la tabulation, DEL et les autres
// caractères de contrôle. Plus permissif que X-4, le motif cacherait l'écart d'une implémentation qui en écrit un. Les flux se lisent en latin1 :
// un octet égale un caractère.
const BREACHES_F10 = /[^\x20-\x7e]/;

// Ce que le contrat fixe d'une exécution : le code, stdout à l'octet et stderr ligne à ligne. Le texte qui suit `fib: ` est libre
// (F9), mais marqué s'il enfreint F10 (X-4). La valeur d'une durée varie par nature : elle est masquée, et l'algorithme chronométré reste
// (F6). Toute autre ligne se compare au texte.
function project({ code, out, err }: Pick<Run, 'code' | 'out' | 'err'>): string {
  const line = (l: string) => {
    if (l.startsWith('fib: ')) return BREACHES_F10.test(l) ? 'fib: … (hors F10)' : 'fib: …';
    return l.replace(/^(fast|iterative|matrix) [0-9]+\.[0-9]{3} ms$/, '$1 … ms');
  };
  return JSON.stringify([code, out, err.split('\n').map(line)]);
}

// Les implémentations regroupées par projection identique, dans l'ordre du PRD § 5 : les groupes par leur premier membre, et
// chaque groupe dans cet ordre. Aucun groupe n'est la référence, pas même le plus nombreux ni celui de la première implémentation.
function group(projections: ReadonlyMap<string, string>): string[][] {
  const groups = new Map<string, string[]>();
  for (const id of IDS) {
    const p = projections.get(id);
    if (p !== undefined) groups.set(p, [...(groups.get(p) ?? []), id]);
  }
  return [...groups.values()];
}

// Plus d'un groupe : un écart.
const hasGap = (groups: readonly string[][]) => groups.length > 1;

// Le témoin, joué avant chaque recherche. Une projection trop large rendrait la boucle aveugle, une trop étroite la noierait sous
// le texte libre et les durées, et un regroupement qui prendrait une implémentation pour référence en ferait un oracle. Il ne lance
// rien, et le contrat ne le joue pas : HAR-6 interdit au contrat d'importer la boucle. Il se contrôle en le mutant à la main (catalogue § 6, « Le témoin de la boucle différentielle »).
function witness(): void {
  const p = (code: number | null, out: string, err: string) => project({ code, out, err });
  assert.equal(p(2, '', 'fib: a\n'), p(2, '', 'fib: b\n'), 'le texte qui suit fib: est libre (F9)');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: \x1b[31ma\n'), 'mais il est marqué hors F10 : ESC');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: \xe9\n'), 'un octet de plus de 127');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\r\n'), 'un retour chariot');
  assert.equal(p(2, '', 'fib: \xe9\n'), p(2, '', 'fib: \xe8\n'), 'le marqueur tient lieu de texte');
  // F10 se lit par X-4 (catalogue, ligne de traçabilité de F10) : rien que de l'ASCII imprimable, de 0x20 à 0x7E, et `\n`, que le découpage a retiré.
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\tb\n'), 'une tabulation : X-4 la refuse, elle est marquée');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\x07\n'), 'BEL, un caractère de contrôle');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\x7f\n'), 'DEL, que 0x7E borne');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\x00\n'), 'NUL, le premier caractère de contrôle');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\x01\n'), 'SOH, juste après NUL');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\x1f\n'), 'US, le dernier avant l’espace (0x20)');
  assert.equal(p(2, '', 'fib: a\n'), p(2, '', 'fib:  ~\n'), 'les bornes de X-4, l’espace (0x20) et le tilde (0x7E), restent du texte libre');
  assert.equal(p(2, '', 'fib: \xe9\n'), p(2, '', 'fib: \x1b[0m\r\xe8\xe8\n'), 'une infraction à F10 vaut une autre : le marqueur est le même');
  assert.equal(p(2, '', 'fib: \xe9\n'), p(2, '', 'fib: a\tb\x7f\x01\n'), 'la tabulation, DEL et les autres caractères de contrôle comme un octet de plus de 127');
  assert.equal(p(0, '55\n', 'fast 0.010 ms\n'), p(0, '55\n', 'fast 1.250 ms\n'), 'la valeur d’une durée varie (F6)');
  assert.notEqual(p(0, '55\n', 'fast 0.010 ms\n'), p(0, '55\n', 'matrix 0.010 ms\n'), 'mais l’algorithme chronométré compte');
  assert.notEqual(p(0, '55\n', 'fast 0.010 ms\n'), p(0, '55\n', 'fast 0.01 ms\n'), 'et le format : trois décimales');
  assert.notEqual(p(0, '55\n', 'fast 0.010 ms\n'), p(0, '55\n', 'fast 0.010 ms\niterative 0.010 ms\n'), 'et le nombre de lignes');
  assert.notEqual(p(0, '55\n', ''), p(0, '55', ''), 'stdout se compare à l’octet : la fin de ligne');
  assert.notEqual(p(0, '55\n', ''), p(0, '55\n\n', ''), 'une ligne de trop');
  assert.notEqual(p(0, '55\n', ''), p(0, '55 \n', ''), 'une espace de trop');
  assert.notEqual(p(2, '', 'fib: a\n'), p(1, '', 'fib: a\n'), 'le code compte');
  assert.notEqual(p(0, 'x', ''), p(0, '', 'x'), 'stdout et stderr restent distincts');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\nfib: b\n'), 'stderr se compare ligne à ligne : une seule ligne fib: (F9)');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a'), 'stderr aussi garde la fin de ligne (F10)');
  assert.notEqual(p(2, '', 'fib: a\n'), p(2, '', 'fib: a\n\n'), 'stderr : une ligne vide de trop');
  assert.notEqual(p(0, '55\n', 'trace x\n'), p(0, '55\n', 'trace y\n'), 'hors fib: et durées, une ligne se compare au texte');

  const odd = new Map([['go', 'x'], ['rust', 'y'], ['ts', 'y'], ['py', 'y']]);
  assert.deepEqual(group(odd), [['go'], ['rust', 'ts', 'py']], 'un écart rend chaque groupe, la majorité comme les autres');
  const three = new Map([['py', 'y'], ['ts', 'z'], ['rust', 'y'], ['go', 'x']]);
  assert.deepEqual(group(three), [['go'], ['rust', 'py'], ['ts']], 'l’ordre du PRD § 5, non celui des insertions, et aucune référence');
  assert.deepEqual(group(new Map([...three].reverse())), group(three), 'le verdict ne dépend pas de l’ordre des lancements');
  const agree = new Map([['go', 'x'], ['py', 'x']]);
  assert.deepEqual(group(agree), [['go', 'py']], 'un accord est un seul groupe');
  assert.ok(hasGap(group(odd)) && hasGap(group(three)) && !hasGap(group(agree)), 'un écart, c’est plus d’un groupe');

  // Les suites de jetons : de 0 à K, par longueur puis dans l'ordre des jetons, et le niveau K + 1, que rien ne joue, n'est jamais construit.
  // Le niveau suivant se construit par un `map` sur les jetons, une fois par suite du niveau courant : 1 + 2 suites sous K = 2 avec deux jetons.
  let built = 0;
  const counted = new Proxy(['a', 'b'], { get: (target, key, receiver) => (key === 'map' && built++, Reflect.get(target, key, receiver)) });
  assert.deepEqual([...grammar(2, counted)].map(({ args }) => args.join('')), ['', 'a', 'b', 'aa', 'ab', 'ba', 'bb'], 'toutes les suites de 0 à K jetons, par longueur');
  assert.equal(built, 3, 'le niveau K + 1 est construit pour rien');
  assert.deepEqual([...grammar(0, ['a', 'b'])].map(({ args }) => args), [[]], 'K = 0 : la seule suite vide');

  // La sélection : une dimension peut ne rien produire (les descripteurs sous Windows), pas toute la sélection. Une recherche sans entrée sortirait
  // en 0, « sans écart », sans avoir rien cherché.
  assert.throws(() => selection(['descripteurs'], 2, 'win32'), /aucune entrée/, 'une sélection sans entrée doit échouer, non sortir en « sans écart »');
  assert.equal(selection(['descripteurs'], 2, 'linux')[0][1].length, SHORT_ARGS.length, 'les descripteurs jouent les lignes de commande courtes, sous POSIX');
  assert.ok(selection(['descripteurs'], 2, 'linux')[0][1].every((entry) => entry.stderrClosed === true), 'les descripteurs ferment le fd 2 (CLI-V04)');
  assert.deepEqual(selection([], 2, 'win32').map(([name, entries]) => [name, entries.length > 0]), [['grammaire', true], ['valeurs', true], ['environnement', true], ['descripteurs', false]], 'par défaut, une dimension vide ne rougit pas');
  assert.deepEqual(selection(['descripteurs', 'valeurs'], 2, 'win32').map(([name]) => name), ['descripteurs', 'valeurs'], 'une dimension vide parmi d’autres ne rougit pas');

  // Les arguments : chaque erreur d'usage se lève dans parseArgs, donc avant ce témoin, toute extraction et tout build (code 2).
  const refused = (args: string[], message: RegExp, why: string) => assert.throws(() => parseArgs(args), message, why);
  assert.deepEqual(parseArgs([]), { ids: [], dimensions: [], tokens: 2, ref: undefined }, 'sans argument : les implémentations présentes, toutes les dimensions, K = 2');
  assert.equal(parseArgs(['--jetons', '0']).tokens, 0, '--jetons 0 est permis : la seule suite vide');
  assert.equal(parseArgs(['--jetons', '3']).tokens, 3, '--jetons rend son entier');
  for (const value of ['', '-1', '2.5', '1e3', '0x10', ' 3', 'abc', '9007199254740993']) refused(['--jetons', value], /--jetons demande/, `--jetons ${JSON.stringify(value)} accepté`);
  refused(['--jetons'], /--jetons demande .*aucune valeur/, '--jetons sans valeur accepté');
  assert.equal(parseArgs(['--revision', 'HEAD~1']).ref, 'HEAD~1', '--revision rend sa révision');
  for (const value of ['', '-x', '--help']) refused(['--revision', value], /--revision demande/, `--revision ${JSON.stringify(value)} accepté`);
  refused(['--revision'], /--revision demande .*aucune valeur/, '--revision sans valeur accepté');
  assert.deepEqual(parseArgs(['py', 'go', 'py', '--dimension', 'valeurs', '--dimension', 'valeurs']).ids, ['py', 'go'], 'un identifiant répété ne compte qu’une fois');
  refused(['--dimension', 'x'], /--dimension demande/, 'une dimension inconnue acceptée');
  refused(['perl'], /implémentation inconnue/, 'un identifiant inconnu accepté');
  refused(['--bogus'], /option inconnue/, 'une option inconnue acceptée');
  // Une seule implémentation demandée n'a rien à comparer : une erreur d'usage, avant la résolution de --revision, non un échec en code 3 après elle.
  for (const args of [['go'], ['go', 'go'], ['py', '--revision', 'HEAD'], ['--dimension', 'valeurs', 'ts']]) refused(args, /au moins deux implémentations/, `${args.join(' ')} : une seule implémentation acceptée`);
  assert.deepEqual(parseArgs(['go', 'py']).ids, ['go', 'py'], 'deux implémentations suffisent');
}

// Les jetons de la grammaire (PRD § 4, règles 1 à 6) : des valides, des invalides, des ambigus. Toutes les suites de 0 à K jetons.
const TOKENS = ['5', '007', '', 'abc', '-1', '-', '--', '-h', '--help', '--list', '--time', '--algo', '--algo=fast', '--algo=', 'all',
  'matrix', 'FAST', '--bogus', '10001', '5 6'];

function* grammar(k: number, tokens: readonly string[] = TOKENS): Generator<Entry> {
  let sequences: string[][] = [[]];
  for (let length = 0; length <= k; length++) {
    for (const args of sequences) yield { args };
    // Le niveau suivant ne se construit que s'il se joue : celui de K + 1 compterait 20 fois plus de suites que celui de K, pour rien.
    if (length < k) sequences = sequences.flatMap((s) => tokens.map((t) => [...s, t]));
  }
}

// Autour des bornes (F8) et des seuils d'optimisation de l'ADR-0004 : un uint64, mille chiffres, `Decimal` dès 170 000 et `str()`
// au-delà de 200 000 bits de Python (environ 288 086), un million.
const TERMS = [0, 1, 2, 3, 92, 93, 94, 1000, 2000, 4781, 4782, 10_000, 10_001, 169_999, 170_000, 170_001, 288_085, 288_086, 288_087,
  999_999, 1_000_000, 1_000_001];
const LISTS = [0, 1, 2, 93, 94, 2000, 9999, 10_000, 10_001];

function* values(): Generator<Entry> {
  for (const n of TERMS) {
    // `iterative` et `all` restent sous 300 000 : à 1 000 000, Python y passe environ 5 s par lancement (F8).
    for (const algo of n <= 300_000 ? ['fast', 'matrix', 'iterative', 'all'] : ['fast', 'matrix']) yield { args: ['--algo', algo, String(n)] };
  }
  for (const n of LISTS) yield { args: ['--list', String(n)] };
}

const SHORT_ARGS = [['10'], ['--time', '50'], ['--time', '--algo', 'all', '50'], ['--list', '3'], ['--help'], ['--bogus', '5'], []];
const ENVS: NonNullable<LaunchOptions['env']>[] = [
  { LANG: 'fr_CA.UTF-8', LC_ALL: 'fr_CA.UTF-8', LC_NUMERIC: 'fr_CA.UTF-8' },
  { LC_ALL: 'C' },
  { PYTHONIOENCODING: 'utf-16' },
  { PYTHONUTF8: '1' },
  { FORCE_COLOR: '1', CLICOLOR_FORCE: '1', PYTHON_COLORS: '1', NO_COLOR: undefined },
  { NO_COLOR: '1' },
  { TERM: 'dumb' },
  { TZ: 'Pacific/Kiritimati' },
];

function* environment(): Generator<Entry> {
  for (const env of ENVS) for (const args of SHORT_ARGS) yield { args, env };
}

// L'état des descripteurs au lancement : la dimension qui a trouvé l'écart de l'étape 9 (CLI-V04). POSIX seulement.
function* descriptors(platform: NodeJS.Platform): Generator<Entry> {
  if (platform !== 'win32') for (const args of SHORT_ARGS) yield { args, stderrClosed: true };
}

const DIMENSIONS: Record<string, (tokens: number, platform: NodeJS.Platform) => Iterable<Entry>> = {
  grammaire: (tokens) => grammar(tokens),
  valeurs: () => values(),
  environnement: () => environment(),
  descripteurs: (_, platform) => descriptors(platform),
};

// Les entrées à jouer, par dimension : les demandées, ou les quatre dans l'ordre ci-dessus. Une dimension peut ne rien produire (les
// descripteurs sous Windows), mais non toute la sélection : une recherche sans entrée sortirait en 0, « sans écart », sans rien chercher.
function selection(dimensions: readonly string[], tokens: number, platform: NodeJS.Platform = process.platform): [name: string, entries: Entry[]][] {
  const chosen = (dimensions.length ? dimensions : Object.keys(DIMENSIONS)).map((name): [string, Entry[]] => [name, [...DIMENSIONS[name](tokens, platform)]]);
  if (chosen.every(([, entries]) => entries.length === 0)) {
    throw new Error(`la sélection (${chosen.map(([name]) => name).join(', ')}) ne produit aucune entrée sous ${platform} : les descripteurs sont POSIX seulement`);
  }
  return chosen;
}

// Un aperçu pour le diagnostic : la longueur de stdout et ses 24 premiers octets, les 100 premiers de stderr. Il garde le rapport court
// (`--list 10000` écrit 10 Mo) et ne peut pas servir de fichier d'attendus, que la boucle ne lit ni ne compare. Une sortie plus courte s'y lit
// en entier, et c'est voulu : l'écart d'une fin de ligne, d'une espace ou d'un chiffre se voit dans `55\n`, non dans une empreinte, et
// presque tous les écarts tiennent dans une sortie courte. Tronquer toujours les rendrait illisibles.
const preview = (r: Run) => `code ${r.code}, stdout ${r.out.length} o ${JSON.stringify(r.out.slice(0, 24))}, stderr ${JSON.stringify(r.err.slice(0, 100))}`;

const describeEntry = ({ args, env, stderrClosed }: Entry) =>
  [
    `fib ${args.map((a) => JSON.stringify(a)).join(' ')}`.trimEnd(),
    env && `env ${Object.entries(env).map(([k, v]) => (v === undefined ? `-${k}` : `${k}=${v}`)).join(' ')}`,
    stderrClosed && 'fd 2 fermé',
  ]
    .filter(Boolean)
    .join(', ');

const USAGE = 'usage : node contracts/differentiel.ts [go|rust|ts|py …] [--dimension grammaire|valeurs|environnement|descripteurs]… [--jetons K] [--revision REF]';

type Options = { ids: string[]; dimensions: string[]; tokens: number; ref?: string };

// Toute erreur levée ici est une erreur d'usage (code 2, avant le témoin, la résolution de --revision et tout build). `--jetons` veut un
// entier en chiffres décimaux et `--revision` une révision qui ne ressemble pas à une option : lancement.ts les valide, comme pour le banc.
// Une seule implémentation demandée n'a rien à comparer; sans aucune, ce sont les présentes (une seule présente, c'est le code 3 de la recherche).
function parseArgs(args: readonly string[]): Options {
  const ids: string[] = [];
  const dimensions: string[] = [];
  let tokens = 2;
  let ref: string | undefined;
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--dimension') {
      const value = args[++i];
      if (value === undefined || !Object.hasOwn(DIMENSIONS, value)) {
        throw new Error(`--dimension demande ${Object.keys(DIMENSIONS).join(', ')} (reçu : ${shown(value)})`);
      }
      if (!dimensions.includes(value)) dimensions.push(value);
    } else if (arg === '--jetons') {
      const value = args[++i];
      const k = decimalInt(value);
      if (k === undefined) throw new Error(`--jetons demande un entier d’au moins 0 (reçu : ${shown(value)})`);
      tokens = k;
    } else if (arg === '--revision') {
      const value = args[++i];
      ref = revisionArg(value);
      if (ref === undefined) throw new Error(`--revision demande une révision (reçu : ${shown(value)})`);
    } else if (IDS.includes(arg)) {
      if (!ids.includes(arg)) ids.push(arg);
    } else {
      throw new Error(`${arg.startsWith('-') ? 'option' : 'implémentation'} inconnue : ${arg}`);
    }
  }
  if (ids.length === 1) throw new Error(`il faut au moins deux implémentations à comparer (reçu : ${ids[0]})`);
  return { ids, dimensions, tokens, ref };
}

const count = (n: number, word: string) => `${n} ${word}${n > 1 ? 's' : ''}`;

// La recherche : chaque entrée de chaque dimension se lance sur les mêmes implémentations, d'une même racine.
function search({ ids: asked, dimensions, tokens, ref }: Options): number {
  // La sélection d'abord : une sélection sans entrée échoue avant toute résolution, extraction et build.
  const plan = selection(dimensions, tokens);
  // La résolution ensuite : une révision introuvable, ou sans le dossier d'un langage demandé, échoue avant toute extraction.
  const root = ref === undefined ? undefined : extract(resolve(ref, asked), REVISIONS);
  const available = present(root);
  const ids = asked.length ? asked : available;
  const missing = ids.filter((id) => !available.includes(id));
  if (missing.length) throw new Error(`${missing.join(', ')} absente${missing.length > 1 ? 's' : ''}${ref === undefined ? '' : ` de ${ref}`}`);
  if (ids.length < 2) throw new Error(`il faut au moins deux implémentations à comparer (présentes : ${available.join(', ') || 'aucune'})`);
  console.log(`recherche sur ${ids.join(', ')}, ${root === undefined ? 'arbre de travail' : `révision ${ref} (${basename(root).slice(0, 7)})`}`);
  for (const id of ids) build(id, root);

  let entries = 0;
  let gaps = 0;
  for (const [name, list] of plan) {
    let n = 0;
    let g = 0;
    for (const entry of list) {
      n++;
      const runs = new Map<string, Run>();
      for (const id of ids) runs.set(id, launch(id, entry.args, { env: entry.env, stderrClosed: entry.stderrClosed, root }));
      const groups = group(new Map([...runs].map(([id, r]) => [id, project(r)])));
      if (!hasGap(groups)) continue;
      g++;
      console.log(`écart (${name}) : ${describeEntry(entry)}`);
      for (const members of groups) console.log(`  ${members.join(', ').padEnd(18)} ${preview(runs.get(members[0])!)}`);
    }
    console.log(`${name} : ${count(n, 'entrée')}, ${count(g, 'écart')}`);
    entries += n;
    gaps += g;
  }
  console.log(`${count(entries, 'entrée')}, ${count(gaps, 'écart')}, sur ${ids.join(', ')}`);
  return gaps ? 1 : 0;
}

function main(argv: readonly string[]): number {
  let options: Options;
  try {
    options = parseArgs(argv);
  } catch (e) {
    // Avant le témoin et tout build : le code 2 des erreurs d'usage, sur stderr, sans pile.
    console.error(`differentiel: ${(e as Error).message}\n${USAGE}`);
    return 2;
  }
  try {
    witness();
  } catch (e) {
    console.error(`differentiel: le témoin rougit, la recherche n’a pas lieu : ${(e as Error).message}`);
    return 3;
  }
  console.log('témoin : passé');
  try {
    return search(options);
  } catch (e) {
    console.error(`differentiel: ${(e as Error).message}`);
    return 3;
  }
}

// Un Node qui charge ce `.ts` mais n'a pas import.meta.main (23.6 et la suite de la ligne 23, 24.0 et 24.1, ou 22.6 à 22.17 et 23.0 à 23.5
// lancés avec --experimental-strip-types) laisserait la boucle sans effet, en code 0, donc « sans écart » : échouer, en nommant la version
// (le garde-fou du banc, PRD § 5). Le code 3, non l'erreur non rattrapée de Node : son code 1 serait un verdict. Ce garde ne couvre pas un
// Node qui ne charge pas le `.ts` (avant 22.18 sans ce drapeau, 23.0 à 23.5) : il sort de lui-même en 1, ERR_UNKNOWN_FILE_EXTENSION sur
// stderr, avant la première ligne d'ici, et ce 1 n'est pas un verdict. La contrainte du PRD (`^22.18 || >=24.2`) écarte les deux cas.
if (import.meta.main === undefined) {
  console.error('differentiel: la boucle demande Node ^22.18 ou >=24.2 (import.meta.main est absent)');
  process.exitCode = 3;
} else if (import.meta.main) {
  process.exitCode = main(process.argv.slice(2));
}
