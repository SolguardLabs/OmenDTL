import assert from "node:assert/strict";
import test from "node:test";
import { summarizeLiquidity, summarizeReport } from "../../client/report-client.ts";
import { runScenario } from "../helpers/omen.ts";

test("weighted liquidity summary aggregates only the requested asset", () => {
  const summary = summarizeLiquidity(
    [
      {
        id: "a",
        asset: "ousdc",
        reserve: 1_000,
        free_liquidity: 300,
        exposure: 500,
        min_buffer: 100,
      },
      { id: "b", asset: "ousdc", reserve: 500, free_liquidity: 100, exposure: 250, min_buffer: 50 },
      { id: "c", asset: "oeur", reserve: 7_000, free_liquidity: 0, exposure: 100, min_buffer: 0 },
    ],
    "ousdc",
  );
  assert.deepEqual(summary, {
    asset: "ousdc",
    vaults: 2,
    reserve: 1_500,
    freeLiquidity: 400,
    exposure: 750,
    minBuffer: 150,
    aggregateCoverageBps: 20_000,
    freeShareBps: 2_666,
  });
});

test("summary rejects missing assets and invalid amounts", () => {
  const positions = [
    { id: "a", asset: "ousdc", reserve: -1, free_liquidity: 0, exposure: 0, min_buffer: 0 },
  ];
  assert.throws(() => summarizeLiquidity(positions, "ousdc"), /non-negative/);
  assert.throws(() => summarizeLiquidity(positions, "oeur"), /not present/);
  assert.throws(
    () => summarizeReport({ protocol: "other", vaults: [] }, "ousdc"),
    /invalid OmenDTL/,
  );
});

test("client reads the public CLI contract", () => {
  const report = runScenario("operator-day");
  const summary = summarizeReport(report, "ousdc");
  assert.equal(summary.vaults, 4);
  assert.ok(summary.reserve > 0);
  assert.equal(summary.aggregateCoverageBps, null);
  const active = summarizeReport(runScenario("forecast-cycle"), "ousdc");
  assert.ok(active.aggregateCoverageBps !== null);
});
