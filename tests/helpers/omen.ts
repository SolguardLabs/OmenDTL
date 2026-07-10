import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const exeName = process.platform === "win32" ? "omendtl.exe" : "omendtl";
const defaultBin = join(root, "out", exeName);

export type AmountBucket = {
  asset: string;
  amount: number;
};

export type VaultReport = {
  id: string;
  asset: string;
  region: string;
  reserve: number;
  forecast_committed: number;
  reservation_held: number;
  settlement_debt: number;
  pending_in: number;
  pending_out: number;
  min_buffer: number;
  free_liquidity: number;
  exposure: number;
  coverage_ratio_bps: number;
  priority: number;
  strategy: string;
  status: string;
};

export type RouteReport = {
  id: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  market: string;
  base_demand: number;
  observed_demand: number;
  forecasted_demand: number;
  open_demand: number;
  forecast_remaining: number;
  reserved: number;
  settled: number;
  failed: number;
  released_variance: number;
  priority: number;
  status: string;
};

export type ForecastReport = {
  id: string;
  cycle: string;
  route: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  basis_demand: number;
  forecast_amount: number;
  committed_amount: number;
  reserved_amount: number;
  settled_amount: number;
  released_amount: number;
  failed_amount: number;
  confidence_bps: number;
  status: string;
};

export type ReservationReport = {
  id: string;
  cycle: string;
  forecast: string;
  route: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  owner: string;
  kind: string;
  amount: number;
  fee: number;
  settlement_due: number;
  status: string;
};

export type SettlementReport = {
  id: string;
  reservation: string;
  route: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  amount: number;
  fee: number;
  net_amount: number;
  status: string;
};

export type DemandUpdateReport = {
  id: string;
  cycle: string;
  route: string;
  previous_demand: number;
  next_demand: number;
  forecast_basis: number;
  variance: number;
  released_to_vault: number;
  confidence_bps: number;
  status: string;
};

export type OmenReport = {
  lab: string;
  scenario: string;
  network_id: string;
  clock: number;
  state_digest: string;
  assets: Array<Record<string, unknown>>;
  accounts: Array<Record<string, unknown>>;
  vaults: VaultReport[];
  routes: RouteReport[];
  cycles: Array<Record<string, unknown>>;
  forecasts: ForecastReport[];
  reservations: ReservationReport[];
  settlements: SettlementReport[];
  demand_updates: DemandUpdateReport[];
  withdrawals: Array<Record<string, any>>;
  views: Array<Record<string, any>>;
  batches: Array<Record<string, any>>;
  settlement_runs: Array<Record<string, any>>;
  totals: Record<string, AmountBucket[]>;
  risk: Record<string, number>;
  invariants: Record<string, boolean>;
  events: Array<Record<string, any>>;
  notes: string[];
};

export function binaryPath(): string {
  return process.env.OMEN_BIN ?? defaultBin;
}

export function ensureBuilt(): void {
  if (process.env.OMEN_BIN) return;
  if (existsSync(defaultBin)) return;
  const result = spawnSync(process.execPath, ["scripts/build.mjs"], {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (result.status !== 0) {
    throw new Error(["build failed", result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n"));
  }
}

export function runBinary(args: string[], expectSuccess = true) {
  ensureBuilt();
  const result = spawnSync(binaryPath(), args, {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (expectSuccess && result.status !== 0) {
    throw new Error(
      [`omendtl ${args.join(" ")} failed`, result.stdout.trim(), result.stderr.trim()]
        .filter(Boolean)
        .join("\n"),
    );
  }
  return result;
}

export function listScenarios(): string[] {
  return runBinary(["--list"]).stdout.trim().split(/\r?\n/).filter(Boolean);
}

export function runScenario(name: string): OmenReport {
  const result = runBinary(["scenario", name]);
  return JSON.parse(result.stdout) as OmenReport;
}

export function validateScenario(name: string): string {
  return runBinary(["validate", name]).stdout.trim();
}

export function byId<T extends { id: string }>(items: T[], id: string): T {
  const item = items.find((entry) => entry.id === id);
  assert.ok(item, `missing id ${id}`);
  return item;
}

export function bucket(entries: AmountBucket[] | undefined, asset: string): number {
  assert.ok(entries, `missing bucket list for ${asset}`);
  const item = entries.find((entry) => entry.asset === asset);
  assert.ok(item, `missing asset ${asset}`);
  return item.amount;
}

export function events(report: OmenReport, kind: string): Array<Record<string, any>> {
  return report.events.filter((event) => event.kind === kind);
}

export function assertDigest(value: unknown): void {
  assert.equal(typeof value, "string");
  assert.match(value as string, /^[0-9a-f]{32}$/);
}

export function assertCommon(report: OmenReport, scenario: string): void {
  assert.equal(report.lab, "OmenDTL");
  assert.equal(report.scenario, scenario);
  assert.equal(report.network_id, "omen-local-liquidity");
  assertDigest(report.state_digest);
  assert.ok(report.assets.length >= 3);
  assert.ok(report.accounts.length >= 4);
  assert.ok(report.vaults.length >= 6);
  assert.ok(report.routes.length >= 4);
  assert.ok(Array.isArray(report.events));
  assert.equal(report.invariants.vaults_non_negative, true);
  assert.equal(report.invariants.accounts_non_negative, true);
  assert.equal(report.invariants.route_links_valid, true);
  assert.equal(report.invariants.forecast_links_valid, true);
  assert.equal(report.invariants.reservation_links_valid, true);
  assert.equal(report.invariants.settlement_links_valid, true);
  assert.equal(report.invariants.cycle_links_valid, true);
  assert.equal(report.invariants.withdrawals_asset_matched, true);
  assert.equal(report.invariants.forecast_accounting_non_negative, true);
  assert.equal(report.invariants.routes_within_forecast_envelope, true);
}

