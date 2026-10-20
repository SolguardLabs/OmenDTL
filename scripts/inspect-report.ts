import { readFileSync } from "node:fs";
import { summarizeReport } from "../client/report-client.ts";

const [reportPath, asset] = process.argv.slice(2);
if (!reportPath || !asset) {
  console.error("Uso: npm run inspect -- <reporte.json> <asset>");
  process.exitCode = 2;
} else {
  try {
    const report: unknown = JSON.parse(readFileSync(reportPath, "utf8"));
    console.log(JSON.stringify(summarizeReport(report, asset), null, 2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
