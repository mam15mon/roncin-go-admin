import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';

const source = fileURLToPath(new URL('../../server/openapi.yaml', import.meta.url));
const target = fileURLToPath(
  new URL('../config/openapi.generated.json', import.meta.url),
);
const document = YAML.parse(await readFile(source, 'utf8'));

await writeFile(target, `${JSON.stringify(document, null, 2)}\n`, 'utf8');
