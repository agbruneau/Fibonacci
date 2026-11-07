// Tests de garde de la CLI, joués dans le processus par run(args, io) (hors seams, catalogue § 6), sauf PERF-TS-9 et PERF-TS-11, qui lancent la source de la CLI (`ts/src/cli.ts`) dans un sous-processus, et PERF-TS-12, qui joue run(args) sous l'Io du processus, flux captés.
// Ils protègent ce que le test de contrat ne peut pas éprouver; la grammaire, les bornes et les octets de sortie lui restent.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { closeSync, mkdtempSync, openSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { terms } from '../../contracts/oracle.ts';
import { algos as table, fib, type Algo } from './fib.ts';
import { run, type Io } from './cli.ts';

// Un Io d'essai : chaque écriture est enregistrée, flux par flux. Sans substitut, fib est le vrai et l'horloge reste figée.
function harness(substitutes: Partial<Pick<Io, 'fib' | 'clock'>> = {}) {
  const stdout: [text: string, encoding: string][] = [];
  const stderr: string[] = [];
  const io: Io = {
    fib,
    clock: () => 0,
    stdout: { write: (text, encoding) => stdout.push([text, encoding]) },
    stderr: { write: (text) => stderr.push(text) },
    ...substitutes,
  };
  return { io, stdout, stderr };
}

// PERF-TS-6 : l'écriture unique de stdout (ADR-0004). Une écriture par ligne coûte cher sur --list, et le contrat ne voit pas les appels.
const SINGLE_WRITE: [args: string[], expected: () => string][] = [
  [['--list', '300'], () => terms().slice(0, 301).join('\n') + '\n'],
  [['--time', '--algo', 'all', '300'], () => terms()[300] + '\n'],
];

for (const [args, expected] of SINGLE_WRITE) {
  test(`PERF-TS-6 : ${args.join(' ')} écrit tout stdout en un seul appel, en latin1`, () => {
    const { io, stdout } = harness();
    assert.equal(run(args, io), 0);
    assert.deepEqual(stdout, [[expected(), 'latin1']]);
  });
}

// PERF-TS-7 : F3. Le seul usage d'un fib substitué : le vrai terme, plus 1 pour un algorithme, tour à tour à chacune des trois places.
// Aucun cas CLI ne peut provoquer une discordance (catalogue § 6), et le test ne recalcule aucun terme.
for (const bad of ['fast', 'iterative', 'matrix'] as const) {
  test(`PERF-TS-7 : si ${bad} diffère des deux autres, --time --algo all 10 rend 3, sans stdout et sans durée`, () => {
    const { io, stdout, stderr } = harness({ fib: (n, algo) => fib(n, algo) + (algo === bad ? 1n : 0n) });
    assert.equal(run(['--time', '--algo', 'all', '10'], io), 3);
    assert.deepEqual(stdout, [], 'aucune écriture sur stdout');
    assert.equal(stderr.length, 1, 'un seul appel à stderr');
    assert.match(stderr[0], /^fib: [^\n]*\n$/);
    assert.doesNotMatch(stderr[0], /\d\.\d{3} ms/, 'une durée (F6) glissée dans le message, même sur sa ligne');
  });
}

// PERF-TS-8 : F6 (catalogue § 6). L'algorithme nommé s'exécute, une ligne par algorithme exécuté, dans l'ordre, en un seul appel
// à stderr, et chaque durée est l'écart de deux lectures de l'horloge autour de son calcul.
// Ce premier test n'avance l'horloge que dans fib : l'étendue de la fenêtre est gardée par le second test de chaque entrée, plus bas
// (ni la conversion décimale du terme ni l'écriture de stdout). L'analyse des arguments, elle aussi hors fenêtre (PRD F6), reste à la
// revue de code : `parse` n'offre aucun point d'injection.
// Le fib d'essai délègue au vrai et fait avancer l'horloge de son coût. Des coûts en fractions binaires, donc des différences
// exactes; l'horloge ne part pas de 0, pour qu'une lecture absolue se voie.
const COST: Record<Algo, [ms: number, line: string]> = {
  fast: [0.5, 'fast 0.500 ms\n'],
  iterative: [1.25, 'iterative 1.250 ms\n'],
  matrix: [2.125, 'matrix 2.125 ms\n'],
};

const EXECUTED: [args: string[], algos: Algo[]][] = [
  [['10'], ['fast']],
  [['--algo', 'iterative', '10'], ['iterative']],
  [['--algo', 'matrix', '10'], ['matrix']],
  [['--algo', 'all', '10'], ['fast', 'iterative', 'matrix']],
];

for (const [args, algos] of EXECUTED) {
  test(`PERF-TS-8 : --time ${args.join(' ')} exécute ${algos.join(' puis ')} et rapporte le coût de chacun, en un seul appel à stderr`, () => {
    let now = 100;
    const calls: Algo[] = [];
    const { io, stderr } = harness({
      clock: () => now,
      fib: (n, algo) => {
        calls.push(algo);
        now += COST[algo][0];
        return fib(n, algo);
      },
    });
    assert.equal(run(['--time', ...args], io), 0);
    assert.deepEqual(calls, algos, 'les algorithmes exécutés, dans l’ordre');
    assert.deepEqual(stderr, [algos.map((a) => COST[a][1]).join('')]);
  });
}

// La fenêtre du chronomètre ne couvre ni la conversion décimale du terme ni l'écriture de stdout (F6). Le fib d'essai rend aux
// trois algorithmes un même objet, donc ils concordent (compute compare par !==), dont la conversion en texte avance l'horloge de
// 8 ms; l'écriture de stdout l'avance de 16 ms. Les durées attendues sont les seuls coûts du calcul, COST.
// Rouge sous le mutant `timed(durations, io, algo, () => String(io.fib(n, algo)))` dans compute (cli.ts), qui ajoute 8 ms à chaque
// durée (vert avant cette garde). L'avance de 16 ms garde de même un chronomètre qui engloberait l'écriture; la structure actuelle
// de run (écriture après compute) ne laisse pas de mutant d'une ligne qui l'éprouve.
for (const [args, algos] of EXECUTED) {
  test(`PERF-TS-8 : --time ${args.join(' ')} ne mesure ni la conversion décimale du terme ni l’écriture de stdout`, () => {
    let now = 100;
    const term = {
      toString: () => {
        now += 8;
        return '55';
      },
    } as unknown as bigint;
    const { io, stdout, stderr } = harness({
      clock: () => now,
      fib: (_n, algo) => {
        now += COST[algo][0];
        return term;
      },
    });
    const slowWrite: Io = {
      ...io,
      stdout: {
        write: (text, encoding) => {
          now += 16;
          return io.stdout.write(text, encoding);
        },
      },
    };
    assert.equal(run(['--time', ...args], slowWrite), 0);
    assert.deepEqual(stdout, [['55\n', 'latin1']]);
    assert.deepEqual(stderr, [algos.map((a) => COST[a][1]).join('')]);
  });
}

// --list n'appelle pas fib : sa durée vient de l'horloge seule. Elle avance de 0,5 ms à chaque lecture, donc la durée est
// exactement l'écart de deux lectures, et la ligne ne peut pas être écrite en dur.
test('PERF-TS-8 : --time --list 10 rapporte iterative, et sa durée est l’écart de deux lectures de l’horloge', () => {
  let now = 100;
  const { io, stderr } = harness({ clock: () => (now += 0.5) });
  assert.equal(run(['--time', '--list', '10'], io), 0);
  assert.equal(now, 101, 'deux lectures');
  assert.deepEqual(stderr, ['iterative 0.500 ms\n']);
});

// L'horloge est appelée en méthode de l'Io : une horloge à état, qui lit this, fonctionne. Détachée, elle lèverait.
test('PERF-TS-8 : l’horloge est appelée en méthode de l’Io, this compris', () => {
  const { io, stderr } = harness();
  const stateful = Object.assign(io, {
    now: 100,
    clock(this: { now: number }) {
      return (this.now += 0.5);
    },
  });
  assert.equal(run(['--time', '10'], stateful), 0);
  assert.deepEqual(stderr, ['fast 0.500 ms\n']);
});

// PERF-TS-9 : la CLI ne charge pas `performance` (ADR-0004, #21). L'horloge est lue avant l'ouverture de stdout, et le premier
// accès à performance charge alors 11 modules : perf_hooks, neuf internal/perf/… et internal/histogram. timed lit l'horloge
// même sans --time : la garde éprouve donc `10` comme `--time 10`, et les deux branches de compute (--list et un terme).
// Elle joue la source de la CLI dans un sous-processus : celui de test a pu charger perf_hooks lui-même. Un module de `--import`
// y écrit process.moduleLoadList sur stderr à la sortie; stdout, le code et la ligne de durée restent ceux de la CLI.
// Seuls perf_hooks et internal/perf/performance sont interdits : eux seuls viennent du global. Les autres internal/perf/…
// ne prouvent rien : utils est chargé au démarrage de Node, performance_entry et observe par net dès que stdout est un tube.
// Le témoin vérifie que le crochet et le prédicat voient bien ces deux noms sous `performance.now()`.
// Rouge sous le mutant `clock: () => performance.now()` dans processIo (cli.ts), ou sous un import de node:perf_hooks.
const root = fileURLToPath(new URL('../..', import.meta.url));
const MARK = 'moduleLoadList:';
const HOOK =
  'data:text/javascript,' +
  encodeURIComponent(
    `import{writeSync}from'node:fs';process.on('exit',()=>writeSync(2,${JSON.stringify(MARK)}+JSON.stringify(process.moduleLoadList)+'\\n'))`,
  );
const onlyPerformance = (modules: string[]) => modules.filter((m) => /\b(perf_hooks|internal\/perf\/performance)$/.test(m));

// Lance node avec le crochet; rend stdout, stderr sans la liste du crochet, la liste et le temps mural en ms. C'est le lanceur de PERF-TS-9 et de PERF-TS-11.
// `status` est le code attendu (0 par défaut). `stdout` est un descripteur de fichier qui reçoit stdout à la place d'un tube (PERF-TS-11) : le
// `stdout` rendu est alors null, et l'appelant lit le fichier. stdin est ignoré; stderr reste un tube, qui porte la liste du crochet.
// NODE_OPTIONS est retiré : un préchargement ou un débogueur hérité ajouterait des modules ou du texte sur stderr.
// L'avertissement du retrait des types, que certains Node 22 écrivent sur stderr, est tu pour la même raison.
function launch(args: string[], { stdout, status = 0 }: { stdout?: number; status?: number } = {}) {
  const start = process.hrtime.bigint();
  const r = spawnSync(process.execPath, ['--disable-warning=ExperimentalWarning', '--import', HOOK, ...args], {
    cwd: root,
    encoding: 'latin1',
    env: { ...process.env, NODE_OPTIONS: undefined },
    stdio: ['ignore', stdout ?? 'pipe', 'pipe'],
    // Filet contre une CLI qui boucle, comme celui du contrat.
    timeout: 300_000,
  });
  const wall = Number(process.hrtime.bigint() - start) / 1e6;
  assert.ifError(r.error);
  assert.equal(r.status, status, r.stderr);
  const at = r.stderr.lastIndexOf(MARK);
  assert.notEqual(at, -1, 'le crochet n’a rien écrit sur stderr');
  return { stdout: r.stdout, stderr: r.stderr.slice(0, at), modules: JSON.parse(r.stderr.slice(at + MARK.length)) as string[], wall };
}

const CLI = join(root, 'ts', 'src', 'cli.ts');
const LIST_10 = '0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n55\n';
const NO_PERF: [args: string[], stdout: string, stderr: RegExp][] = [
  [['10'], '55\n', /^$/],
  [['--time', '10'], '55\n', /^fast \d+\.\d{3} ms\n$/],
  [['--time', '--list', '10'], LIST_10, /^iterative \d+\.\d{3} ms\n$/],
  [['--time', '--algo', 'all', '10'], '55\n', /^fast \d+\.\d{3} ms\niterative \d+\.\d{3} ms\nmatrix \d+\.\d{3} ms\n$/],
];

for (const [args, stdout, stderr] of NO_PERF) {
  test(`PERF-TS-9 : ${args.join(' ')} ne charge ni perf_hooks ni internal/perf/performance, et la sortie reste celle de la CLI`, () => {
    const r = launch([CLI, ...args]);
    assert.equal(r.stdout, stdout);
    assert.match(r.stderr, stderr);
    assert.deepEqual(onlyPerformance(r.modules), []);
  });
}

test('PERF-TS-9 : témoin, le crochet voit performance charger perf_hooks et internal/perf/performance', () => {
  const loaded = onlyPerformance(launch(['-e', 'performance.now()']).modules);
  assert.ok(loaded.some((m) => m.endsWith('perf_hooks')), `perf_hooks absent de ${loaded.join(', ')}`);
  assert.ok(loaded.some((m) => m.endsWith('internal/perf/performance')), `internal/perf/performance absent de ${loaded.join(', ')}`);
});

// PERF-TS-8 et le contrat ne voient pas l'unité de l'horloge réelle : la première est factice, le second ne lit que le format.
// La durée d'un calcul de quelques dizaines de ms (iterative 100000, environ 85 ms) doit être en millisecondes : strictement
// positive, au plus le temps mural du sous-processus, et au moins son vingtième (le démarrage de Node en fait le reste; la
// borne basse, 20 fois sous la valeur attendue, ne rougit pas sous charge).
// Rouge sous les mutants `/ 1e6` retiré (nanosecondes, au-dessus du temps mural), `/ 1e3` (microsecondes, idem) et `/ 1e9`
// (secondes : 0,085, sous le vingtième du temps mural).
test('PERF-TS-9 : l’horloge réelle rend des millisecondes, de --time --algo iterative 100000 à son temps mural', () => {
  const r = launch([CLI, '--time', '--algo', 'iterative', '100000']);
  const m = /^iterative (\d+\.\d{3}) ms\n$/.exec(r.stderr);
  assert.ok(m, `ligne de durée inattendue : ${JSON.stringify(r.stderr)}`);
  const ms = Number(m[1]);
  assert.ok(ms > 0, `durée ${ms} ms non positive`);
  assert.ok(ms <= r.wall, `durée ${ms} ms au-delà du temps mural ${r.wall.toFixed(3)} ms`);
  assert.ok(ms >= r.wall / 20, `durée ${ms} ms sous le vingtième du temps mural ${r.wall.toFixed(3)} ms`);
});

// PERF-TS-11 : les accesseurs paresseux de processIo (cli.ts, `get stdout()` et `get stderr()`, ADR-0004) : le chemin d'erreur n'ouvre pas
// process.stdout. Le contrat ne le voit pas : il lit la sortie, non les modules chargés. Quand stdout est un fichier, process.stdout se
// construit en SyncWriteStream et charge internal/fs/sync_write_stream; avec deux tubes, stdout et stderr chargent les mêmes modules,
// donc un fichier de tmpdir() comme stdout est essentiel. `--bogus 5` sort en 2 sans rien écrire sur stdout : le module ne doit pas se charger.
// Cette garde joue le `launch` de PERF-TS-9 (même crochet moduleLoadList, même NODE_OPTIONS retiré), avec un fichier comme stdout et le code 2 attendu.
// Le témoin, sous les mêmes stdio, vérifie que `process.stdout` charge ce module et que `0` ne le charge pas : sans lui, la garde serait aveugle.
// Rouge sous le mutant `stdout: process.stdout, stderr: process.stderr` dans processIo (cli.ts), et sous `stdout: process.stdout` seul;
// `stderr: process.stderr` seul la laisse verte : stderr est un tube ici, et le chemin d'erreur l'ouvre de toute façon.
const SYNC_WRITE_STREAM = /\binternal\/fs\/sync_write_stream$/;

// `launch` avec un fichier de tmpdir() comme stdout, retiré ensuite; rend son résultat, dont `stdout` est le contenu du fichier.
function launchToFile(args: string[], status = 0) {
  const dir = mkdtempSync(join(tmpdir(), 'fib-perf-ts-11-'));
  const file = join(dir, 'stdout');
  const fd = openSync(file, 'w');
  try {
    const r = launch(args, { stdout: fd, status });
    return { ...r, stdout: readFileSync(file, 'latin1') };
  } finally {
    closeSync(fd);
    rmSync(dir, { recursive: true, force: true });
  }
}

test('PERF-TS-11 : --bogus 5 sort en 2, laisse le fichier de stdout vide et ne charge pas internal/fs/sync_write_stream', () => {
  const r = launchToFile([CLI, '--bogus', '5'], 2);
  assert.equal(r.stdout, '', 'stdout vide');
  assert.match(r.stderr, /^fib: [^\n]*\n$/, 'la CLI a bien écrit sa ligne d’erreur');
  assert.deepEqual(r.modules.filter((m) => SYNC_WRITE_STREAM.test(m)), []);
});

test('PERF-TS-11 : témoin, avec un fichier comme stdout, process.stdout charge internal/fs/sync_write_stream et 0 ne le charge pas', () => {
  const loaded = launchToFile(['-e', 'process.stdout']).modules.filter((m) => SYNC_WRITE_STREAM.test(m));
  assert.ok(loaded.length > 0, 'internal/fs/sync_write_stream absent des modules de process.stdout');
  assert.deepEqual(launchToFile(['-e', '0']).modules.filter((m) => SYNC_WRITE_STREAM.test(m)), []);
});

// PERF-TS-12 : le `fib` de processIo (cli.ts) est celui de la bibliothèque (F2, F6). PERF-TS-6 à 8 le remplacent par un Io d'essai et
// ne le voient pas; PERF-TS-9 lance la vraie CLI et n'y voit, le plus souvent mais sans garantie, que `iterative` remplacé par `fast` ou `matrix`
// (l'unité de la durée, plus haut : sa borne basse lâche laisse passer un calcul de remplacement que la charge ralentit). Le contrat non plus ne le voit pas : les trois algorithmes rendent les mêmes valeurs.
// `run(args)` sans Io joue sous processIo : chaque ligne de durée doit avoir appelé l'entrée de la table `algos` (fib.ts, la même
// instance de module que celle de cli.ts) qu'elle nomme. Des espions y remplacent les trois entrées : ils notent leur nom et rendent
// 0n sans calculer, comme ceux de PERF-TS-10, donc n = 1000000 (la borne F8) ne coûte rien. Chaque entrée de EXECUTED est jouée à
// n = 10 et à n = 1000000; stdout reçoit le seul `0\n`, et les entrées appelées sont les noms des lignes de durée, dans l'ordre.
// Les flux du processus sont captés le temps de l'appel, synchrone : rien n'y passe jusqu'au lanceur, que l'isolation par processus
// ou `--test-isolation=none` ou `node ts/src/cli.test.ts` hébergent. Aucune durée n'est jugée (l'horloge de processIo est la réelle) :
// les lignes ne sont lues que par leur premier mot.
// Rouge sous les mutants `fib: (n, algo) => fib(n, algo === 'matrix' ? 'fast' : algo)` (les tests et le contrat y restaient verts
// avant cette garde), `fib: (n, algo) => fib(n, n > 10 && algo === 'matrix' ? 'fast' : algo)` (à n = 1000000 seulement) et
// `fib: (n) => fib(n, 'fast')` (rouge aussi, d'ordinaire, sous PERF-TS-9, sans garantie) dans processIo (cli.ts).
// Survivant connu, laissé à la revue de code : un aiguillage qui dépend de n hors de ces deux indices, tel que
// `n > 10 && n < 1000000 && algo === 'matrix' ? 'fast' : algo`.
const SPIED_INDEXES = [10, 1000000];

// Remplace `write` sur un flux du processus et note les textes reçus, sans rien lui transmettre. `write` vient d'ordinaire du prototype
// du flux, non de lui : restaurer, c'est alors supprimer la propriété posée, non la réaffecter, pour que le flux retrouve la valeur
// héritée. Seule une propriété propre d'avant l'appel est réaffectée.
function capture(stream: NodeJS.WriteStream) {
  const own = Object.hasOwn(stream, 'write');
  const original = stream.write;
  const chunks: unknown[] = [];
  stream.write = ((chunk: unknown) => {
    chunks.push(chunk);
    return true;
  }) as typeof stream.write;
  return {
    chunks,
    restore() {
      if (own) stream.write = original;
      else Reflect.deleteProperty(stream, 'write');
    },
  };
}

// run(args) sous l'Io du processus, ses deux flux captés. Rien n'est assertionné ici : les flux sont rendus avant le premier assert.
function runOnProcessIo(args: string[]) {
  const out = capture(process.stdout);
  const err = capture(process.stderr);
  try {
    return { code: run(args), stdout: out.chunks, stderr: err.chunks };
  } finally {
    err.restore();
    out.restore();
  }
}

for (const n of SPIED_INDEXES) {
  for (const [args, algos] of EXECUTED) {
    const options = args.slice(0, -1); // le dernier argument est l'indice, que n remplace
    test(`PERF-TS-12 : sous l’Io du processus, --time ${[...options, n].join(' ')} appelle ${algos.join(' puis ')} dans la table algos`, () => {
      // Conversion explicite : `algos` est en lecture seule par le type; seuls les espions (PERF-TS-10, PERF-TS-12) y écrivent, et chacun restaure la table dans finally.
      const spies = table as Record<Algo, (n: number) => bigint>;
      const original = { ...spies };
      const called: Algo[] = [];
      let result;
      try {
        for (const name of Object.keys(COST) as Algo[]) {
          spies[name] = () => {
            called.push(name);
            return 0n;
          };
        }
        result = runOnProcessIo(['--time', ...options, String(n)]);
      } finally {
        Object.assign(spies, original);
      }
      assert.equal(result.code, 0);
      assert.deepEqual(result.stdout, ['0\n']);
      assert.equal(result.stderr.length, 1, 'un seul appel à stderr');
      const named = [...String(result.stderr[0]).matchAll(/^(\S+) \S+ ms$/gm)].map((m) => m[1]);
      assert.deepEqual(named, algos, 'les lignes de durée');
      assert.deepEqual(called, named, 'les entrées de la table appelées, dans l’ordre des lignes de durée');
    });
  }
}
