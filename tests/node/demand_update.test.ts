import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, events, runScenario } from "../helpers/omen.ts";

test("demand updates refresh observed route state inside the active window", () => {
  const report = runScenario("demand-update");
  assertCommon(report, "demand-update");
  assert.equal(report.demand_updates.length, 2);
  assert.ok(events(report, "demand.updated").length >= 2);

  const route = byId(report.routes, "route-atlas-boreal");
  const update = report.demand_updates.find((entry) => entry.route === route.id);
  assert.ok(update);
  assert.equal(route.observed_demand, update.next_demand);
  assert.ok(update.confidence_bps >= 6500);
});

test("variance release is reported as route and vault accounting", () => {
  const report = runScenario("demand-update");
  const update = report.demand_updates.find((entry) => entry.route === "route-atlas-boreal");
  assert.ok(update);
  assert.ok(update.variance > 0);
  assert.ok(update.released_to_vault > 0);

  const route = byId(report.routes, "route-atlas-boreal");
  assert.equal(route.released_variance, update.released_to_vault);
});

