export function consoleAuthDisabled(): boolean {
  return (process.env.DONTSTOP_CONSOLE_AUTH ?? "").trim().toLowerCase() === "off";
}
