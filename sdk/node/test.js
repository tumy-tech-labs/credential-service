import { Client } from './index.js';

async function run() {
  const client = new Client({
    baseUrl: 'http://localhost:0',
    fetchImpl: async () => ({ ok: true, status: 200, text: async () => '{}' })
  });

  await client.verify('dummy');
  console.log('node sdk smoke test passed');
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
