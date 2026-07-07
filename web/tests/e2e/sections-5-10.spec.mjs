import { expect, test } from "@playwright/test";

async function createIncidentWithEvidence(page, prefix) {
  const title = `${prefix} ${Date.now()}`;
  await page.goto("/");
  await page.getByLabel("Incident title", { exact: true }).fill(title);
  await page.getByRole("button", { name: "Create incident" }).click();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();
  await page.getByPlaceholder("Share an update…").fill("HTTP 500 followed the latest deployment.");
  await page.getByRole("button", { name: "Post update" }).click();
  return title;
}

test("Compose provides context collection and evidence-linked AI synthesis", async ({ page }) => {
  await createIncidentWithEvidence(page, "Context and AI");

  await page.getByLabel("Add optional content").selectOption("context");
  await page.getByLabel("Context recipe").selectOption({ label: "Service health v1" });
  await page.getByRole("button", { name: "Preview recipe" }).click();
  await expect(page.locator(".context-preview")).toContainText("Preview only — not published");
  await expect(page.locator(".context-preview")).toContainText("2 planned operations");
  await expect(page.locator("article.post")).toHaveCount(1);
  await page.getByRole("button", { name: "Collect context" }).click();
  const collection = page.locator(".context-collection").last();
  await expect(collection).toContainText("Collection complete");
  await expect(collection).toContainText("2 snapshots · 0 failures");
  await page.getByRole("button", { name: "Close optional content" }).click();

  await page.getByLabel("Incident agent").selectOption({ label: "Incident synthesizer · local-deterministic" });
  await page.getByRole("button", { name: "Activate agent" }).click();
  await page.getByLabel("Agent task").fill("Summarize the failure and collected context");
  await page.getByRole("button", { name: "Run synthesis" }).click();
  await expect(page.getByText(/Summarize the failure and collected context · succeeded/)).toBeVisible();
  await expect(page.getByText("AI summary", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Accept AI proposal" }).click();
  await expect(page.getByText(/accepted · .* evidence links/)).toBeVisible();
});

test("Compose runs a guided workflow and the traceable remediation sequence", async ({ page }) => {
  await createIncidentWithEvidence(page, "Workflow and remediation");

  await page.getByLabel("Workflow definition").selectOption({ label: "Guided mitigation v1" });
  await page.getByRole("button", { name: "Start workflow" }).click();
  const workflow = page.locator(".workflow-run").last();
  await expect(workflow).toContainText("Investigate evidence");
  const investigateStep = workflow.locator(".workflow-step", { hasText: "Investigate evidence" });
  await investigateStep.getByRole("button", { name: "Start step" }).click();
  await investigateStep.getByRole("button", { name: "Complete step" }).click();
  await expect(workflow).toContainText("Apply approved mitigation");

  await page.getByRole("button", { name: "Start disposable investigation" }).click();
  const investigation = page.locator(".investigation").last();
  await expect(investigation).toContainText("investigating");
  await investigation.getByRole("button", { name: "Reproduce defect" }).click();
  await expect(investigation).toContainText("diagnostic: Reproduced the reported defect");
  await investigation.getByRole("button", { name: "Authorise patch workspace" }).click();
  await expect(investigation).toContainText("Patch workspace agent/incident-");
  await investigation.getByRole("button", { name: "Apply patch" }).click();
  await investigation.getByRole("button", { name: "Verify patch" }).click();
  await investigation.getByRole("button", { name: "Capture staging evidence" }).click();
  await expect(investigation).toContainText("browser: Captured allowed staging evidence");
  await investigation.getByRole("button", { name: "Create traceable pull request" }).click();
  await expect(investigation).toContainText("GitHub pull request #1");
  await investigation.getByRole("button", { name: "Destroy sandbox" }).click();
  await expect(investigation).toContainText("destroyed");
});

test("incidents can be related and split with provenance and independent access", async ({ page }) => {
  const first = await createIncidentWithEvidence(page, "Combined incident");
  const second = `Known recurrence ${Date.now()}`;
  await page.getByLabel("Incident title", { exact: true }).fill(second);
  await page.getByRole("button", { name: "Create incident" }).click();

  await page.getByLabel("Related incident").selectOption({ label: second });
  await page.getByLabel("Relationship type").selectOption("recurrence-of");
  await page.getByRole("button", { name: "Link incident" }).click();
  await expect(page.locator(".relationship-row")).toContainText(`recurrence-of · ${second}`);

  await page.getByLabel("Add optional content").selectOption("knowledge");
  await page.getByLabel("Knowledge search").fill("HTTP 500");
  await page.getByRole("button", { name: "Search knowledge" }).click();
  await expect(page.locator(".search-result", { hasText: `post · ${first}` })).toBeVisible();
  await page.getByRole("button", { name: "Close optional content" }).click();
  const derivedResponse = page.waitForResponse(response => response.request().method() === "POST" && response.url().endsWith("/derive"));
  await page.locator("article.post", { hasText: "HTTP 500 followed" }).getByRole("button", { name: "Repost to selected incident" }).click();
  const derived = await (await derivedResponse).json();
  expect(derived.derivedFromPostId).toBeTruthy();
  await page.getByRole("button", { name: `unclassified ${second}` }).click();
  await expect(page.getByText("HTTP 500 followed the latest deployment.")).toBeVisible();
  await page.getByRole("button", { name: `unclassified ${first}` }).click();
  await page.getByLabel("Related incident").selectOption({ label: second });

  const splitTitle = `Checkout sub-incident ${Date.now()}`;
  await page.getByLabel("Split incident title").fill(splitTitle);
  const splitResponse = page.waitForResponse(response =>
    response.request().method() === "POST" && response.url().endsWith("/split")
  );
  await page.getByRole("button", { name: "Split feed into incident" }).click();
  const payload = await (await splitResponse).json();
  expect(payload.copiedPosts).toHaveLength(1);
  expect(payload.copiedPosts[0].derivedFromPostId).toBeTruthy();
  expect(payload.relationship.kind).toBe("parent-of");

  await page.getByRole("button", { name: `unclassified ${splitTitle}` }).click();
  await expect(page.getByText("HTTP 500 followed the latest deployment.")).toBeVisible();
  await expect(page.locator(".relationship-row", { hasText: `parent-of · ${first}` })).toBeVisible();

  await page.getByLabel("Development principal").fill("bob");
  await page.getByRole("button", { name: "Switch" }).click();
  await expect(page.getByRole("button", { name: `unclassified ${splitTitle}` })).toHaveCount(0);
});

test("approved AI detection records its gates and supports false-alarm cancellation", async ({ page }) => {
  await page.goto("/");
  const title = `Detected saturation ${Date.now()}`;
  const headers = { "X-Principal-ID": "alice" };
  const agents = await (await page.request.get("/api/v1/workspaces/acme/agents", { headers })).json();
  const detectionResponse = await page.request.post("/api/v1/workspaces/acme/ai-incident-detections", {
    headers,
    data: { detectorId: agents.items[0].id, title, severity: "SEV-2", trigger: "automated evaluation", rule: "compose-demo-detection-policy-v1", confidence: 0.95, confidenceGate: 0.8, minimumSeverity: "SEV-4", supportingEvidence: [`operator-visible signal: ${title}`] },
  });
  const result = await detectionResponse.json();
  expect(result.created).toBe(true);
  expect(result.incident.detection.confidence).toBe(0.95);
  expect(result.incident.detection.confidenceGate).toBe(0.8);
  expect(result.incident.detection.rule).toBe("compose-demo-detection-policy-v1");
  expect(result.incident.detection.supportingEvidence).toHaveLength(1);

  await page.reload();
  await page.getByRole("button", { name: `SEV-2 ${title}` }).click();
  await expect(page.getByText(/SEV-2 · open · Opened by /)).toBeVisible();
  await page.getByRole("button", { name: "Cancel false alarm" }).click();
  await expect(page.getByText(/SEV-2 · cancelled · Opened by /)).toBeVisible();
  await expect(page.getByRole("button", { name: "Cancel false alarm" })).toHaveCount(0);
});

test("running incidents explicitly migrate template versions and retain prior snapshots", async ({ page }) => {
  const headers = { "X-Principal-ID": "alice" };
  const v1Response = await page.request.post("/api/v1/workspaces/acme/incident-templates", {
    headers,
    data: { name: "Migration template", description: "v1", defaultSeverity: "SEV-3", defaultScope: ["service"], closureChecklist: [{ id: "verify", label: "Verify", completed: false }] },
  });
  expect(v1Response.status()).toBe(201);
  const v1 = await v1Response.json();
  await page.goto("/");
  const title = `Template migration ${Date.now()}`;
  await page.getByLabel("Incident title", { exact: true }).fill(title);
  await page.getByLabel("Incident template").selectOption(v1.id);
  await page.getByRole("button", { name: "Create incident", exact: true }).click();

  const v2Response = await page.request.post(`/api/v1/workspaces/acme/incident-templates/${v1.id}/versions`, {
    headers,
    data: { name: "Migration template", description: "v2", defaultSeverity: "SEV-1", defaultScope: ["checkout", "production"], closureChecklist: [{ id: "recover", label: "Recovery verified", completed: false }] },
  });
  expect(v2Response.status()).toBe(201);
  await page.reload();
  await page.getByRole("button", { name: `SEV-3 ${title}` }).click();
  await expect(page.getByText("Migration template v1 · 0 prior snapshots")).toBeVisible();
  await page.getByLabel("Incident configuration template").selectOption(v1.id);
  await page.getByRole("button", { name: "Migrate configuration" }).click();
  await expect(page.getByText("Migration template v2 · 1 prior snapshots")).toBeVisible();
  await expect(page.getByText("checkout, production", { exact: true })).toBeVisible();
  await expect(page.getByText(/SEV-1 · open · Opened by /)).toBeVisible();
});
