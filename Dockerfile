FROM oven/bun:1

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .

# MCP server (stdio) — override with `bun run dev` for Next.js
CMD ["bun", "mcp/server.ts"]
