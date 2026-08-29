export const MCP_VERSION = "0.1.1";

export function versionsMatch(runtimeVersion: string, mcpVersion = MCP_VERSION): boolean {
  const a = runtimeVersion.replace(/-.*$/, "").split(".");
  const b = mcpVersion.replace(/-.*$/, "").split(".");
  if (a[0] !== b[0]) {
    return false;
  }
  if (a[0] === "0") {
    return a[1] === b[1];
  }
  return true;
}
