import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, events, runScenario } from "../helpers/omen.ts";

test("liquidity window composes refresh, withdrawal and settlement events", () => {
  const report = runScenario("liquidity-window");
  assertCommon(report, "liquidity-window");
  assert.equal(report.withdrawals.length, 1);
  assert.equal(report.withdrawals[0].status, "completed");
  assert.ok(report.withdrawals[0].amount > 0);
  assert.ok(events(report, "withdrawal.completed").length, 1);
  assert.ok(report.settlement_runs.length >= 1);
  assert.ok(bucket(report.totals.withdrawal_debits, "ousdc") > 0);
});

test("operator day covers full forecast and settlement lifecycle", () => {
  const report = runScenario("operator-day");
  assertCommon(report, "operator-day");
  assert.ok(report.views.length >= 4);
  assert.ok(report.batches[0].accepted.length >= 4);
  assert.ok(report.risk.finalized_settlements >= 2);
  assert.equal(report.risk.frozen_settlements, 0);
  assert.ok(report.cycles.every((cycle: any) => cycle.status === "closed"));
});

