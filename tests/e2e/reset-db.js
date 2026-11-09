const { Client } = require("pg");

async function main() {
  const connectionString = process.env.E2E_DATABASE_URL;
  if (!connectionString) {
    console.error("E2E_DATABASE_URL is required");
    process.exit(1);
  }
  const client = new Client({ connectionString });
  await client.connect();
  try {
    await client.query(
      "TRUNCATE admin_sessions, admin_users, provider_accounts, provider_models, providers CASCADE"
    );
  } finally {
    await client.end();
  }
}

main().catch(error => {
  console.error(error);
  process.exit(1);
});
