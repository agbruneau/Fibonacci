// Lancement d'une implémentation : le seul lieu des commandes de build et de lancement (PRD § 5). Le test de contrat et
// le banc d'essai l'appellent tous deux : le banc mesure ainsi ce que le contrat vérifie. La boucle différentielle, un outil
// de recherche hors du contrat, s'en sert aussi, et le banc comme elle résolvent puis extraient une révision (`resolve`, `extract`).
import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, renameSync, rmSync, symlinkSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

// La racine de ce dépôt : la racine par défaut de tout lancement. Les révisions de l'A/B du banc et de la boucle y sont aussi rangées.
export const ROOT = fileURLToPath(new URL('..', import.meta.url));
const exe = process.platform === 'win32' ? '.exe' : '';

// Lancement depuis une racine : celle de ce dépôt, ou celle d'une révision extraite (l'A/B du banc). Go, Rust et TS passent
// par un build fait une fois, pour mesurer le programme et non son démarrage. Go et Rust tirent leur binaire de la racine
// (chemin absolu); TS et Python s'y lancent comme dossier courant (chemins relatifs). Python : `-I -S` saute `site` et PYTHON*,
// sans changer la sortie de la CLI. tsc 7 est lancé par le point d'entrée JS du paquet : `.bin/tsc` n'est pas exécutable sans
// shell sous Windows. `build` s'exécute lui aussi depuis la racine, donc `node_modules` doit s'y trouver.
const IMPLS: Record<string, { dir: string; build?: string[]; cmd: (root: string) => string[] }> = {
  go: {
    dir: 'go',
    build: ['go', '-C', 'go', 'build', '-o', `../.build/fib${exe}`, './cmd/fib'],
    cmd: (root) => [join(root, '.build', `fib${exe}`)],
  },
  rust: {
    dir: 'rust',
    build: ['cargo', 'build', '-q', '--release', '--manifest-path', 'rust/Cargo.toml'],
    cmd: (root) => [join(root, 'rust', 'target', 'release', `fib${exe}`)],
  },
  ts: {
    dir: 'ts',
    build: [process.execPath, 'node_modules/typescript/bin/tsc', '-p', 'tsconfig.build.json'],
    cmd: () => [process.execPath, '.build/ts/cli.js'],
  },
  py: { dir: 'python', cmd: () => ['python', '-I', '-S', 'python/fib.py'] },
};

function impl(id: string) {
  if (!Object.hasOwn(IMPLS, id)) throw new Error(`implémentation inconnue : ${id}`);
  return IMPLS[id];
}

// Tous les identifiants du PRD § 5, dans cet ordre, que le dossier existe ou non : l'ensemble de HAR-3, qui n'a pas besoin du disque.
export const IDS: readonly string[] = Object.keys(IMPLS);

// Le dossier de l'implémentation à la racine du dépôt (`python` pour `py`) : le banc le cherche dans l'arbre d'une révision avant de l'extraire.
export const dirOf = (id: string): string => impl(id).dir;

// Les identifiants des implémentations dont le dossier existe sous root (par défaut ce dépôt), dans l'ordre du PRD § 5 : ce que le
// contrat et le banc jouent par défaut. Une révision plus ancienne peut ne pas avoir tous les dossiers.
export const present = (root = ROOT): string[] => IDS.filter((id) => existsSync(join(root, IMPLS[id].dir)));

// La commande de lancement de l'implémentation depuis root, sans argument : l'exécutable puis ses arguments fixes. HAR-1 et HAR-5 y comparent l'appel du banc.
export const command = (id: string, root = ROOT): string[] => impl(id).cmd(root);

// Compile Go, Rust ou TypeScript depuis root; rien pour Python. Un échec lève une erreur qui porte la sortie du compilateur.
export function build(id: string, root = ROOT): void {
  const cmd = impl(id).build;
  if (!cmd) return;
  const r = spawnSync(cmd[0], cmd.slice(1), { cwd: root, encoding: 'utf8' });
  if (r.status !== 0) throw new Error(`${cmd.join(' ')} a échoué (code ${r.status}) : ${r.stdout}${r.stderr}${r.error ?? ''}`);
}

// POSIX : `sh` ferme le descripteur 2, puis exécute la commande avec ses arguments inchangés ($0 la commande, $@ le reste).
// spawnSync ne sait pas fermer un fd : stdio 'ignore' ouvre /dev/null, ce qui n'est pas la même chose (vérifié).
export const closeStderr = (argv: readonly string[]): [string, string[]] => ['/bin/sh', ['-c', 'exec "$0" "$@" 2>&-', ...argv]];

export type LaunchOptions = {
  // S'ajoute à l'environnement du processus; une valeur undefined retire la variable (spawnSync l'omet).
  env?: Record<string, string | undefined>;
  // Lance la CLI le fd 2 fermé (CLI-V04), par /bin/sh : POSIX seulement.
  stderrClosed?: boolean;
  // La racine d'où lancer, celle d'une révision extraite pour l'A/B du banc : son binaire (Go, Rust) et son dossier courant. Par défaut, ce dépôt.
  root?: string;
};

// Lance la CLI construite et rend son code, ses deux flux en latin1 (un octet égale un caractère, donc X-2, X-4 et les
// longueurs voient les octets bruts) et la durée du seul spawnSync, que le banc rapporte.
export function launch(id: string, args: readonly string[], { env, stderrClosed = false, root = ROOT }: LaunchOptions = {}) {
  const argv = [...command(id, root), ...args];
  // Sans stderrClosed, la CLI elle-même : le banc ne paie jamais le démarrage de /bin/sh.
  const [file, rest] = stderrClosed ? closeStderr(argv) : [argv[0], argv.slice(1)];
  const options = {
    cwd: root,
    env: { ...process.env, ...env },
    encoding: 'latin1',
    // `--list 10000` écrit environ 10,5 Mo, au-delà du défaut.
    maxBuffer: 1 << 28,
    // Filet contre une CLI qui boucle (PRD § 6).
    timeout: 300_000,
  } as const;
  // Les options sont prêtes avant le chronomètre : il n'entoure que spawnSync, comme le banc l'a toujours fait.
  const start = process.hrtime.bigint();
  const r = spawnSync(file, rest, options);
  const ms = Number(process.hrtime.bigint() - start) / 1e6;
  if (r.error) throw new Error(`${id} ${args.join(' ')} : ${r.error.message}`, { cause: r.error });
  return { code: r.status, out: r.stdout, err: r.stderr, ms };
}

// Les seuls chemins du build (PRD § 5) : une révision extraite n'emporte ni CLAUDE.md ni .claude/, qu'une session lancée sous
// .build/ lirait comme des instructions. package.json y reste : son "type": "module" fait de .build/ts/cli.js un module ES.
const BUILD_PATHS = ['go/', 'rust/', 'ts/', 'python/', 'package.json', ':(glob)tsconfig*.json'];

// Où s'extraient les révisions du banc (l'A/B) et de la boucle différentielle : sous .build/, ignoré par git. `--nettoyer` les supprime.
export const REVISIONS = join(ROOT, '.build', 'revisions');

// La validation des valeurs d'option que le banc (`--runs`, `--contre`) et la boucle différentielle (`--jetons`, `--revision`) lisent de même.
// Chacune rend la valeur valide ou undefined, et l'appelant écrit l'erreur d'usage avec le nom de son option. Un entier ne s'écrit qu'en chiffres
// décimaux, et sûr : Number() accepte aussi '', ' 3', '1e3' et '0x10', et rend NaN, 0 ou une boucle sans fin pour le reste.
export const decimalInt = (value: string | undefined): number | undefined =>
  value !== undefined && /^[0-9]+$/.test(value) && Number.isSafeInteger(Number(value)) ? Number(value) : undefined;
// Une révision ne ressemble pas à une option, que git lirait comme telle (`--help`), et n'est pas vide.
export const revisionArg = (value: string | undefined): string | undefined => (value === undefined || value === '' || value.startsWith('-') ? undefined : value);
// La valeur reçue, telle que l'erreur d'usage la rapporte.
export const shown = (value: string | undefined): string => (value === undefined ? 'aucune valeur' : JSON.stringify(value));

// Résout ref en SHA, puis vérifie que la révision a le dossier de chaque langage de ids : avant toute extraction et tout build, car un
// langage absent de ref ne se découvre sinon qu'au milieu de l'A/B, après les langages qui le précèdent. Elle ne lit la révision que par
// `git rev-parse` et `git ls-tree`. Lève une erreur d'une ligne, que le banc et la boucle écrivent sans pile.
export function resolve(ref: string, ids: readonly string[]): string {
  const git = (...args: string[]) => spawnSync('git', args, { cwd: ROOT, encoding: 'utf8' });
  const resolved = git('rev-parse', '--verify', '--quiet', `${ref}^{commit}`);
  if (resolved.status !== 0) throw new Error(`révision introuvable : ${ref}`);
  const sha = resolved.stdout.trim();
  const tree = git('ls-tree', '-d', '-z', '--name-only', sha);
  if (tree.status !== 0) throw new Error(`git ls-tree ${sha} a échoué (code ${tree.status}) : ${tree.stderr}${tree.error ?? ''}`);
  const dirs = tree.stdout.split('\0');
  const absent = ids.filter((id) => !dirs.includes(dirOf(id)));
  if (absent.length) throw new Error(`${absent.join(', ')} : dossier absent de ${ref}`);
  return sha;
}

// Extrait la révision sha (rendue par `resolve`) sous revisions/<sha> et rend sa racine. Une fois : si le dossier existe, il est réutilisé,
// avec ses binaires déjà construits. L'extraction passe par un index à part (GIT_INDEX_FILE) et un dossier provisoire renommé à la fin :
// ni l'index ni la liste des worktrees ne changent, et une extraction interrompue ne laisse pas un dossier à moitié rempli.
export function extract(sha: string, revisions: string): string {
  const git = (args: string[], env: Record<string, string> = {}, input?: string) => {
    const r = spawnSync('git', args, { cwd: ROOT, encoding: 'utf8', env: { ...process.env, ...env }, input });
    if (r.status !== 0) throw new Error(`git ${args.join(' ')} a échoué (code ${r.status}) : ${r.stderr}${r.error ?? ''}`);
    return r.stdout;
  };
  const dest = join(revisions, sha);
  if (existsSync(dest)) return dest;
  const part = `${dest}.part`;
  const index = `${dest}.index`;
  rmSync(part, { recursive: true, force: true });
  rmSync(index, { force: true });
  mkdirSync(part, { recursive: true });
  const env = { GIT_INDEX_FILE: index };
  git(['read-tree', sha], env);
  git(['checkout-index', '-z', '--stdin', `--prefix=${part}/`], env, git(['ls-files', '-z', '--', ...BUILD_PATHS], env));
  rmSync(index);
  // tsc et les @types y sont cherchés dans node_modules : la révision emprunte ceux de ce dépôt (jonction sous Windows).
  symlinkSync(join(ROOT, 'node_modules'), join(part, 'node_modules'), 'junction');
  renameSync(part, dest);
  return dest;
}
