export type VaultPosition = {
  id: string;
  asset: string;
  reserve: number;
  free_liquidity: number;
  exposure: number;
  min_buffer: number;
};

export type LiquiditySummary = {
  asset: string;
  vaults: number;
  reserve: number;
  freeLiquidity: number;
  exposure: number;
  minBuffer: number;
  aggregateCoverageBps: number | null;
  freeShareBps: number;
};

function safeAmount(value: unknown, field: string): number {
  if (!Number.isSafeInteger(value) || (value as number) < 0) {
    throw new TypeError(`${field} must be a non-negative safe integer`);
  }
  return value as number;
}

export function summarizeLiquidity(
  positions: readonly VaultPosition[],
  asset: string,
): LiquiditySummary {
  if (!asset.trim()) throw new TypeError("asset is required");
  const selected = positions.filter((position) => position.asset === asset);
  if (selected.length === 0) throw new RangeError(`asset not present: ${asset}`);

  let reserve = 0;
  let freeLiquidity = 0;
  let exposure = 0;
  let minBuffer = 0;
  for (const position of selected) {
    reserve += safeAmount(position.reserve, `${position.id}.reserve`);
    freeLiquidity += safeAmount(position.free_liquidity, `${position.id}.free_liquidity`);
    exposure += safeAmount(position.exposure, `${position.id}.exposure`);
    minBuffer += safeAmount(position.min_buffer, `${position.id}.min_buffer`);
  }
  for (const [field, value] of Object.entries({ reserve, freeLiquidity, exposure, minBuffer })) {
    safeAmount(value, field);
  }
  return {
    asset,
    vaults: selected.length,
    reserve,
    freeLiquidity,
    exposure,
    minBuffer,
    aggregateCoverageBps: exposure === 0 ? null : Math.floor((reserve * 10_000) / exposure),
    freeShareBps: reserve === 0 ? 0 : Math.floor((freeLiquidity * 10_000) / reserve),
  };
}

export function summarizeReport(report: unknown, asset: string): LiquiditySummary {
  if (typeof report !== "object" || report === null)
    throw new TypeError("report must be an object");
  const value = report as { protocol?: unknown; vaults?: unknown };
  if (value.protocol !== "OmenDTL" || !Array.isArray(value.vaults)) {
    throw new TypeError("invalid OmenDTL report");
  }
  return summarizeLiquidity(value.vaults as VaultPosition[], asset);
}
