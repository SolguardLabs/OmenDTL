import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, bucket, runScenario } from "../helpers/omen.ts";

test("reservation batch assigns forecast liquidity across active routes", () => {
  const report = runScenario("reservation");
  assertCommon(report, "reservation");
  assert.equal(report.batches.length, 1);
  assert.ok(report.reservations.length >= 4);
  assert.ok(report.reservations.every((reservation) => reservation.status === "queued"));
  assert.ok(report.reservations.every((reservation) => reservation.amount > reservation.fee));
  assert.ok(bucket(report.totals.reserved_routes, "ousdc") > 0);
});

test("route state tracks reserved capacity without settling immediately", () => {
  const report = runScenario("reservation");
  const route = byId(report.routes, "route-atlas-boreal");
  const reservation = report.reservations.find((item) => item.route === route.id);
  assert.ok(reservation);
  assert.equal(route.settled, 0);
  assert.equal(route.failed, 0);
  assert.equal(route.reserved, reservation.amount);
  assert.ok(route.forecast_remaining >= 0);
});

