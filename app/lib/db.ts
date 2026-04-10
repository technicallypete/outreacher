import { Pool } from "pg";

// Direct postgres connection for NextAuth adapter.
// Uses the nextauth schema first so @auth/pg-adapter finds its tables
// without schema qualification, then falls back to app schema.
const pool = new Pool({
  connectionString:
    process.env.DATABASE_URL ?? "postgresql://app:app@localhost:5432/outreacher",
  options: "-c search_path=nextauth,app",
});

export default pool;
