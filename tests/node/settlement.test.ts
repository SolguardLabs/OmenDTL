import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, events, runScenario } from "../helpers/omen.ts";

test("settlement scenario finalizes due reservations and records fees", () => {
  const report = runScenario("settlement");
  assertCommon(report, "settlement");
  assert.ok(report.settlements.length >= 4);
  assert.ok(report.settlements.some((settlement) => settlement.status === "finalized"));
  assert.ok(report.reservations.some((reservation) => reservation.status === "completed"));
  assert.ok(events(report, "settlement.finalized").length >= 1);
  assert.ok(bucket(report.totals.finalized_fees, "ousdc") > 0);
});

test("target vault reserves increase by net settlement amount", () => {
  const report = runScenario("settlement");
  const finalized = report.settlements.filter((settlement) => settlement.status === "finalized");
  assert.ok(finalized.length > 0);
  for (const settlement of finalized) {
    assert.ok(settlement.net_amount > 0);
    assert.equal(settlement.amount - settlement.fee, settlement.net_amount);
  }
  assert.equal(report.risk.frozen_settlements, 0);
});

