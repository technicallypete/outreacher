import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { SSEClientTransport } from "@modelcontextprotocol/sdk/client/sse.js";

const MCP_URL = process.env.MCP_URL ?? "http://localhost:3001";

export type MCPContent = { type: string; text: string };
export type MCPToolResult = { content: MCPContent[]; isError?: boolean };

/**
 * Connects to the MCP SSE server, calls one tool, closes the connection,
 * and returns the raw result. All API routes use this instead of importing
 * the MCP SDK directly.
 */
export async function callTool(
  name: string,
  args: Record<string, unknown> = {}
): Promise<MCPToolResult> {
  const transport = new SSEClientTransport(new URL(`${MCP_URL}/sse`));
  const client = new Client({ name: "outreacher-app", version: "0.1.0" });

  await client.connect(transport);
  try {
    const result = await client.callTool({ name, arguments: args });
    return result as MCPToolResult;
  } finally {
    await client.close();
  }
}

/**
 * Parses the text content from an MCP tool result. Throws if the result
 * is an error or has no text content.
 */
export function parseResult<T>(result: MCPToolResult): T {
  if (result.isError) {
    const msg = result.content[0]?.text ?? "MCP tool error";
    throw new Error(msg);
  }
  const text = result.content[0]?.text;
  if (!text) throw new Error("MCP tool returned no content");
  return JSON.parse(text) as T;
}
