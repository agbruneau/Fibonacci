// Banc d'essai des CLI (ADR-0004) : médiane du temps total du processus et du calcul rapporté par --time.
// Usage : node contracts/bench.ts [go|rust|ts|py …] [--runs N] [--contre REF] | --nettoyer. Compile puis lance par contracts/lancement.ts, comme le contrat.
// --contre REF : A/B apparié, la révision REF (A) contre l'arbre de travail (B), lancés en alternance (le travail d'un démarrage, ADR-0004).
// Un nombre pair de paires seulement (100 par défaut) : un --runs impair est arrondi au pair supérieur, et dit sur stderr.
// --nettoyer : vide .build/revisions/, où l'A/B extrait ses révisions, sans suivre le lien vers node_modules qu'elles portent.
import { existsSync, lstatSync, readdirSync, rmSync, unlinkSync } from 'node:fs';
import { basename, join } from 'node:path';
import { build, decimalInt, extract, IDS, launch, present, resolve, REVISIONS, revisionArg, shown } from './lancement.ts';

const SCENARIOS: [name: string, args: string[]][] = [
  ['petit (10)', ['10']],
  ['terme (100000, fast)', ['100000']],
  ['terme (1000000, fast)', ['1000000']],
  // Le seul indice impair : la branche impaire du dernier bit de fast (les autres indices sont pairs).
  ['terme (999999, fast)', ['999999']],
  ['matrix 100000', ['--algo', 'matrix', '100000']],
  ['iterative 100000', ['--algo', 'iterative', '100000']],
  ['all 100000', ['--algo', 'all', '100000']],
  ['liste 10000', ['--list', '10000']],
];

// La médiane de toute mesure du banc (A, B et B − A). Un nombre pair : la moyenne des deux éléments centraux, car l'élément haut
// seul biaiserait chaque résumé vers le haut (HAR-5).
const median = (xs: readonly number[]) => {
  const s = [...xs].sort((a, b) => a - b);
  const h = s.length >> 1;
  return s.length % 2 === 1 ? s[h] : (s[h - 1] + s[h]) / 2;
};

// Une exécution avec --time : temps total du processus, et somme des durées qu'il rapporte. Exportée pour le harnais.
// root : la racine d'une révision extraite (A/B), par défaut ce dépôt.
export function measure(id: string, args: readonly string[], root?: string): { wall: number; calc: number } {
  // La commande de l'implémentation, puis --time et les arguments, sans enveloppeur ni stderrClosed (HAR-1).
  const r = launch(id, ['--time', ...args], { root });
  if (r.code !== 0) throw new Error(`${id} ${args.join(' ')} : code ${r.code}, ${r.err}`);
  const calc = r.err.split('\n').filter(Boolean).reduce((sum, line) => sum + Number(line.split(' ')[1]), 0);
  return { wall: r.ms, calc };
}

// Le tour i d'un A/B : A puis B aux tours pairs, B puis A aux impairs, pour qu'aucun des deux ne suive toujours l'autre
// (le second lancement profite des caches du premier). Rend toujours [A, B]. Exportée pour le harnais (HAR-5).
export function pair<T>(i: number, a: () => T, b: () => T): [a: T, b: T] {
  if (i % 2 === 0) {
    const x = a();
    return [x, b()];
  }
  const y = b();
  return [a(), y];
}

// Résume des paires [A, B] : les médianes de A et de B, la médiane des différences B − A (négative quand B est plus rapide), et le
// nombre de paires où B est strictement plus rapide. La médiane des différences, non la différence des médianes : la dérive de la
// machine touche A et B d'un même tour, et seule la différence par paire l'annule. Exportée pour le harnais (HAR-5).
export function paired(pairs: readonly (readonly [a: number, b: number])[]): { a: number; b: number; d: number; faster: number } {
  const d = pairs.map(([a, b]) => b - a);
  return { a: median(pairs.map(([a]) => a)), b: median(pairs.map(([, b]) => b)), d: median(d), faster: d.filter((x) => x < 0).length };
}

// Une mesure d'A/B qui échoue porte son côté : « go 10 : code 1 » ne dit pas, seul, si la révision ou l'arbre de travail est en cause.
const side = <T>(label: string, f: () => T): T => {
  try {
    return f();
  } catch (e) {
    throw new Error(`côté ${label} : ${(e as Error).message}`, { cause: e });
  }
};

// Un tour d'A/B sur un scénario, en temps total : A depuis la racine de la révision, B depuis ce dépôt. Si A partait aussi de
// ce dépôt, l'A/B comparerait l'arbre de travail à lui-même. Exportée pour le harnais (HAR-5).
export const abRound = (i: number, id: string, args: readonly string[], root: string): [a: number, b: number] =>
  pair(
    i,
    () => side(`A (la révision REF, ${root})`, () => measure(id, args, root).wall),
    () => side('B (l’arbre de travail)', () => measure(id, args).wall),
  );

// La politique de mesure que le banc et l'A/B partagent : `one(i, args)` mesure un scénario au tour i, et rounds rend, par scénario, ses
// mesures des tours 1 à runs. Le tour 0 réchauffe les caches et s'écarte; les scénarios s'entrelacent dans chaque tour pour que la
// dérive de la machine les touche tous. Exportée pour le harnais (HAR-5).
export function rounds<T>(runs: number, one: (i: number, args: string[]) => T): T[][] {
  const kept = SCENARIOS.map((): T[] => []);
  for (let i = 0; i <= runs; i++) {
    SCENARIOS.forEach(([, args], s) => {
      const x = one(i, args);
      if (i > 0) kept[s].push(x);
    });
  }
  return kept;
}

// Les paires [A, B] gardées d'un A/B, par scénario : rounds, dont chaque rendez-vous joue abRound avec l'indice du tour (qui règle l'ordre)
// et la racine de la révision. Un nombre PAIR de paires, refusé sinon avant tout lancement : l'ordre alterne ici, et sur un nombre impair un côté
// part le premier dans une paire de plus, où la médiane des différences retombe avec l'effet d'ordre que l'alternance annule (ADR-0004).
// `main` arrondit au pair supérieur ce que l'utilisateur demande; un autre appelant doit donner un pair. Exportée pour le harnais (HAR-5),
// sans build ni extraction.
export function abPairs(id: string, runs: number, root: string): [a: number, b: number][][] {
  if (!(runs >= 2 && runs % 2 === 0)) throw new Error(`un A/B joue un nombre pair de paires, au moins 2 (reçu : ${runs})`);
  return rounds(runs, (i, args) => abRound(i, id, args, root));
}

// Supprime les révisions extraites et rend le nombre d'entrées supprimées. Chaque révision porte un lien (jonction sous Windows) vers le
// node_modules de ce dépôt, et `Remove-Item -Recurse` de Windows PowerShell 5.1 peut le traverser et vider sa cible : le lien est donc
// détaché d'abord (unlink retire le lien, jamais sa cible), puis le dossier tombe sans rien à traverser. Rien d'autre sous .build/ n'est touché.
function clean(revisions: string): number {
  if (!existsSync(revisions)) return 0;
  const entries = readdirSync(revisions);
  for (const entry of entries) {
    const link = join(revisions, entry, 'node_modules');
    try {
      if (lstatSync(link).isSymbolicLink()) unlinkSync(link);
    } catch (e) {
      // Une entrée sans lien (ENOENT), ou un fichier de reste comme <sha>.index (ENOTDIR), n'a rien à détacher.
      if (!['ENOENT', 'ENOTDIR'].includes((e as NodeJS.ErrnoException).code ?? '')) throw e;
    }
  }
  rmSync(revisions, { recursive: true, force: true });
  return entries.length;
}

const USAGE = 'usage : node contracts/bench.ts [go|rust|ts|py …] [--runs N] [--contre REF] | --nettoyer';

// Les exécutions par scénario sans --runs : 11 pour le banc, des médianes qui suffisent à situer un temps. Un A/B en veut 100 paires, soit
// 200 lancements par scénario, l'ordre de grandeur des 6 tours de 201 lancements qui ont tranché l'horloge de #21 : l'erreur de la médiane
// des différences décroît en 1/√n, donc d'un facteur 3 environ sur 11 paires, ce qui sépare un écart de quelques pour cent du bruit.
// Un nombre PAIR : l'ordre alterne, et sur un nombre impair de paires un côté part le premier une fois de plus, la médiane retombe dans ce
// groupe et l'effet d'ordre que l'alternance annule revient (+0,27 ms sur 101 paires, 0,00 sur 100, en simulation). ADR-0004.
const RUNS = 11;
const RUNS_AB = 100;

// Les identifiants demandés (aucun : les implémentations présentes), le nombre d'exécutions par scénario, et la révision REF d'un A/B
// (absente sans `--contre`, donc sans clé). Toute erreur levée ici est une erreur d'usage. `--runs` veut des chiffres décimaux seulement
// (sans quoi cell() découvre trop tard un NaN ou un 0) et `--contre` une révision qui ne ressemble pas à une option : lancement.ts les
// valide pour le banc et pour la boucle (`decimalInt`, `revisionArg`). Tout autre argument est un identifiant du PRD § 5 : un identifiant
// inconnu ou une option mal écrite sont refusés ici, avant tout build. `--nettoyer` vide `.build/revisions/` et s'emploie seul (`clean: true`,
// absent sinon).
export function parseArgs(args: readonly string[]): { runs: number; ids: string[]; ref?: string; clean?: true } {
  if (args.includes('--nettoyer')) {
    if (args.length > 1) throw new Error('--nettoyer s’emploie seul');
    return { runs: RUNS, ids: [], clean: true };
  }
  let runs: number | undefined;
  let ref: string | undefined;
  const ids: string[] = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--contre') {
      const value = args[++i];
      ref = revisionArg(value);
      if (ref === undefined) throw new Error(`--contre demande une révision (reçu : ${shown(value)})`);
      continue;
    }
    if (args[i] !== '--runs') {
      if (!IDS.includes(args[i])) throw new Error(`argument inconnu : ${JSON.stringify(args[i])} (identifiants : ${IDS.join(', ')})`);
      ids.push(args[i]);
      continue;
    }
    const value = args[++i];
    const n = decimalInt(value);
    if (n === undefined || n < 1) throw new Error(`--runs demande un entier d’au moins 1 (reçu : ${shown(value)})`);
    runs = n;
  }
  runs ??= ref === undefined ? RUNS : RUNS_AB;
  return ref === undefined ? { runs, ids } : { runs, ids, ref };
}

const cell = (x: number) => x.toFixed(1).padStart(8);

// Le banc ordinaire : médianes du temps total et du calcul, par langage et par scénario.
function table(ids: string[], runs: number): void {
  console.log(`médianes sur ${runs} exécutions (ms) : total du processus / calcul selon --time`);
  console.log(`${'scénario'.padEnd(22)}${ids.map((id) => `${id.padStart(10)} total   calcul`).join('')}`);
  const rows = SCENARIOS.map(([name]) => name.padEnd(22));
  for (const id of ids) {
    build(id);
    const samples = rounds(runs, (_, args) => measure(id, args));
    samples.forEach((xs, s) => {
      rows[s] += `${cell(median(xs.map((x) => x.wall)))} ${cell(median(xs.map((x) => x.calc)))}   `;
    });
  }
  console.log(rows.join('\n'));
}

// L'A/B apparié : la révision ref, extraite sous rev, contre l'arbre de travail, en temps total. Il ne compare que des temps, jamais
// des sorties : ref n'est pas un attendu (ADR-0001). Comme le banc, il lève si une CLI sort avec un code non nul. Les deux côtés
// se construisent avec les commandes d'aujourd'hui, justes tant qu'elles n'ont pas changé entre ref et l'arbre de travail.
function versus(ids: string[], runs: number, ref: string, rev: string): void {
  console.log(`A/B apparié sur ${runs} paire${runs > 1 ? 's' : ''} (ms, temps total) : A = ${ref} (${basename(rev).slice(0, 7)}), B = l'arbre de travail; ordre alterné, tour 0 écarté`);
  for (const id of ids) {
    build(id);
    build(id, rev);
    const pairs = abPairs(id, runs, rev);
    console.log(`${id.padEnd(22)}${'A'.padStart(8)} ${'B'.padStart(8)} ${'B − A'.padStart(8)} ${'%'.padStart(7)}   B < A`);
    pairs.forEach((p, s) => {
      const e = paired(p);
      console.log(`${SCENARIOS[s][0].padEnd(22)}${cell(e.a)} ${cell(e.b)} ${cell(e.d)} ${((100 * e.d) / e.a).toFixed(1).padStart(7)}   ${e.faster}/${p.length}`);
    });
  }
}

// Le banc : le code de sortie, 2 pour une erreur d'usage et 1 pour une révision inutilisable, l'une et l'autre écrites en une ligne
// `bench: …` sur stderr, sans pile, avant tout build et toute extraction. `revisions` : où se rangent les révisions de l'A/B.
export function main(argv: readonly string[], revisions: string = REVISIONS): number {
  let parsed: ReturnType<typeof parseArgs>;
  try {
    parsed = parseArgs(argv);
  } catch (e) {
    console.error(`bench: ${(e as Error).message}\n${USAGE}`);
    return 2;
  }
  if (parsed.clean) {
    const n = clean(revisions);
    console.log(n ? `${n} entrée${n > 1 ? 's' : ''} supprimée${n > 1 ? 's' : ''} de ${revisions}` : `rien à supprimer sous ${revisions}`);
    return 0;
  }
  const { runs, ref } = parsed;
  const ids = parsed.ids.length ? parsed.ids : present();
  if (ref === undefined) {
    table(ids, runs);
    return 0;
  }
  // Une révision introuvable, un langage absent de REF ou une extraction qui échoue : avant tout build, pour ne pas le payer deux fois.
  let rev: string;
  try {
    rev = extract(resolve(ref, ids), revisions);
  } catch (e) {
    console.error(`bench: ${(e as Error).message}`);
    return 1;
  }
  // Un --runs impair est arrondi au pair supérieur, et dit : sur un nombre impair de paires, un côté part le premier une fois de plus et la
  // médiane des différences garde cet effet d'ordre. Après la lecture de la révision (une erreur reste seule sur stderr), avant tout build.
  const pairs = runs + (runs % 2);
  if (pairs !== runs) console.error(`bench: --runs ${runs} est impair sous --contre, arrondi à ${pairs} paires : A et B doivent partir le premier à égalité`);
  versus(ids, pairs, ref, rev);
  return 0;
}

// Un Node qui charge ce `.ts` mais n'a pas import.meta.main (23.6 et la suite de la ligne 23, 24.0 et 24.1, ou 22.6 à 22.17 et 23.0 à 23.5
// lancés avec --experimental-strip-types) laisserait le banc sans effet, en code 0 : échouer, en nommant la version (le garde-fou de
// ts/src/cli.ts, PRD § 5). Un Node qui ne charge pas le `.ts` du tout (avant 22.18 sans ce drapeau) échoue de lui-même avant ce garde.
// L'import par le test de contrat échoue de même : il demande le même Node.
if (import.meta.main === undefined) throw new Error('le banc demande Node ^22.18 ou >=24.2 (import.meta.main est absent)');
if (import.meta.main) process.exitCode = main(process.argv.slice(2));
