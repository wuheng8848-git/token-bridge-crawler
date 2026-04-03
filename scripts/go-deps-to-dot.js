const readline = require('readline');
const PREFIX = 'token-bridge-crawler/';
const rl = readline.createInterface({ input: process.stdin });
const edges = [];

rl.on('line', (line) => {
  const idx = line.indexOf('|');
  if (idx === -1) return;
  const pkg = line.substring(0, idx);
  const depsStr = line.substring(idx + 1);
  if (!pkg.startsWith(PREFIX)) return;
  const shortPkg = pkg.substring(PREFIX.length);
  if (!depsStr) return;
  for (const dep of depsStr.split(',')) {
    if (dep.startsWith(PREFIX)) {
      const shortDep = dep.substring(PREFIX.length);
      edges.push(`  "${shortPkg}" -> "${shortDep}";`);
    }
  }
});

rl.on('close', () => {
  console.log('digraph crawler_package_deps {');
  console.log('  rankdir=LR;');
  console.log('  node [shape=box, style=filled, fillcolor=lightblue, fontname="Consolas"];');
  console.log('  edge [color=gray40];');
  edges.forEach(e => console.log(e));
  console.log('}');
});
