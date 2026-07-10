import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, events, runScenario } from "../helpers/omen.ts";

test("forecast cycle commits route liquidity before reservations are opened", () => {
  const report = runScenario("forecast-cycle");
  assertCommon(report, "forecast-cycle");
  assert.equal(report.cycles.length, 1);
  assert.equal(report.forecasts.length, 4);
  assert.equal(report.reservations.length, 0);
  assert.ok(events(report, "forecast.committed").length >= 4);

  const committed = bucket(report.totals.forecast_committed, "ousdc");
  const atlas = report.vaults.find((vault) => vault.id === "vault-atlas-ousdc");
  assert.ok(atlas);
  assert.ok(committed > 0);
  assert.ok(atlas.forecast_committed > 0);
  assert.ok(atlas.free_liquidity < atlas.reserve);
});

test("projected view includes route-level forecast profiles", () => {
  const report = runScenario("forecast-cycle");
  const projected = report.views.find((view) => view.mode === "projected");
  assert.ok(projected);
  assert.ok(projected.profiles.some((profile: any) => profile.forecast_committed > 0));
  assert.ok(projected.routes.every((route: any) => route.forecast_remaining >= 0));
});
