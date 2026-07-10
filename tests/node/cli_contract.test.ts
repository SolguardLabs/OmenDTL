import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, listScenarios, runBinary, runScenario, validateScenario } from "../helpers/omen.ts";

test("cli lists deterministic scenario names", () => {
  const scenarios = listScenarios();
  assert.deepEqual(scenarios, [
    "baseline",
    "forecast-cycle",
    "reservation",
    "demand-update",
    "settlement",
    "liquidity-window",
    "operator-day",
  ]);
});

test("baseline report exposes stable json contract", () => {
  const report = runScenario("baseline");
  assertCommon(report, "baseline");
  assert.equal(report.forecasts.length, 0);
  assert.equal(report.reservations.length, 0);
  assert.equal(report.views.length, 1);
  assert.match(validateScenario("baseline"), /^ok baseline [0-9a-f]{32}$/);
});

test("unknown scenario exits with a non-zero status", () => {
  const result = runBinary(["scenario", "missing"], false);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /invalid scenario/);
});

