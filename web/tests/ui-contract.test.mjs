import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const main = readFileSync(new URL("../src/Main.elm", import.meta.url), "utf8");
const css = readFileSync(new URL("../public/app.css", import.meta.url), "utf8");
const html = readFileSync(new URL("../public/index.html", import.meta.url), "utf8");

test("trusted controls have accessible names and use live API state", () => {
  for (const label of ["Current principal", "Block type", "New participant", "Coordination type", "Chronological feed"]) assert.match(main, new RegExp(label));
  assert.match(main, /port apiRequest/);
  assert.match(html, /fetch\(request\.url/);
});

test("critical actions remain available in the responsive interface", () => {
  for (const action of ["Advance to", "Post update", "Save summary", "Add participant"]) assert.match(main, new RegExp(action));
  assert.match(css, /@media \(max-width: 800px\)/);
  assert.doesNotMatch(css, /\.channel-nav\s*\{\s*display:\s*none/);
});
