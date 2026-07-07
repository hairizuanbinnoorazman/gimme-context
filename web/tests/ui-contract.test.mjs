import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const main = readFileSync(new URL("../src/Main.elm", import.meta.url), "utf8");
const css = readFileSync(new URL("../public/app.css", import.meta.url), "utf8");
const html = readFileSync(new URL("../public/index.html", import.meta.url), "utf8");

test("trusted controls have accessible names and use live API state", () => {
  for (const label of ["Development principal", "New participant", "Coordination type", "Chronological feed"]) assert.match(main, new RegExp(label));
  assert.match(main, /port apiRequest/);
  assert.match(html, /fetch\(request\.url/);
});

test("incidents identify their opener without presenting AI incidents as a separate type", () => {
  assert.match(main, /" · Opened by "/);
  assert.doesNotMatch(main, /" · AI detected"/);
  assert.doesNotMatch(main, /Create AI-detected incident/);
});

test("post content uses markdown with fenced code instead of mismatched block choices", () => {
  assert.match(main, /Encode\.string "markdown"/);
  assert.match(main, /String\.startsWith "```"/);
  assert.doesNotMatch(main, /aria-label" "Block type"/);
  assert.match(css, /\.code-block/);
});

test("replies render beneath their parent with one nesting rail", () => {
  assert.match(main, /viewPostThread/);
  assert.match(main, /candidate\.replyToPostId == post\.id/);
  assert.match(css, /\.post-children/);
  assert.doesNotMatch(css, /\.post-reply\s*\{[^}]*margin-left/);
});

test("the single-block composer exposes one post-level reply action", () => {
  assert.match(main, /text "Reply to post"/);
  assert.doesNotMatch(main, /text "Reply to block"/);
});

test("posts show a date-time and optional contribution tools live in the composer", () => {
  assert.match(main, /Html\.time \[ attribute "datetime" post\.createdAt/);
  assert.match(main, /Add optional content/);
  assert.match(main, /AI knowledge base addition/);
  assert.doesNotMatch(main, /incidentState model incident =[\s\S]*h2 \[\] \[ text "Structured coordination"/);
});

test("critical actions remain available in the responsive interface", () => {
  for (const action of ["Advance to", "Post update", "Save summary", "Add participant"]) assert.match(main, new RegExp(action));
  assert.match(css, /@media \(max-width: 800px\)/);
  assert.doesNotMatch(css, /\.channel-nav\s*\{\s*display:\s*none/);
});

test("workflow safety controls are trusted and visible", () => {
  for (const label of ["Workflow projection", "Start workflow", "Stop workflow", "Stop autonomy", "Authorised restart", "Retry"]) assert.match(main, new RegExp(label));
  assert.match(main, /\[ "checklist", "flow" \]/);
});

test("sandbox and pull-request evidence controls are visible", () => {
  for (const label of ["Start disposable investigation", "Read-only checkout", "reproducible evidence blocks", "GitHub pull request", "Destroy sandbox"]) assert.match(main, new RegExp(label));
  assert.match(main, /sha256:/);
});
