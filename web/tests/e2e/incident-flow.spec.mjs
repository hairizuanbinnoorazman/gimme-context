import { expect, test } from "@playwright/test";

const errorsByPage = new WeakMap();

test.beforeEach(async ({ page }) => {
  const browserErrors = [];
  errorsByPage.set(page, browserErrors);
  page.on("console", message => {
    if (message.type() === "error") browserErrors.push(message.text());
  });
  page.on("pageerror", error => browserErrors.push(error.message));
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Gimme Context" })).toBeVisible();
  await expect(page.locator(".top-bar")).toHaveCSS("background-color", "rgb(21, 61, 111)");
  await expect(page.locator(".workspace")).toHaveCSS("display", "grid");
  await expect(page.getByRole("button", { name: "Create incident" })).toBeDisabled();
});

test.afterEach(async ({ page }) => {
  expect(errorsByPage.get(page), "browser console errors").toEqual([]);
});

test("a human creates an incident and posts an investigation update", async ({ page }) => {
  const title = `Checkout latency ${Date.now()}`;
  await page.getByLabel("Incident title").fill(title);
  await page.getByLabel("Incident severity").selectOption("SEV-2");
  await page.getByLabel("Incident scope").fill("checkout, production");
  await page.getByRole("button", { name: "Create incident" }).click();

  const incident = page.getByRole("button", { name: `SEV-2 ${title}` });
  await expect(incident).toBeVisible();
  await incident.click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  await expect(page.getByText("checkout, production", { exact: true })).toBeVisible();

  await page.getByPlaceholder("Share an update…").fill("Latency started after the latest checkout deployment.");
  await page.getByLabel("Block type").selectOption("status");
  await page.getByRole("button", { name: "Post update" }).click();
  await expect(page.getByText("Latency started after the latest checkout deployment.")).toBeVisible();
});

test("operators record and accept a decision", async ({ page }) => {
  const title = `Payments decision ${Date.now()}`;
  await page.getByLabel("Incident title").fill(title);
  await page.getByRole("button", { name: "Create incident" }).click();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();

  await page.getByLabel("Coordination type").selectOption("decision");
  await page.getByPlaceholder("Statement, task, or question").fill("Roll back the payments deployment.");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByText("Roll back the payments deployment.")).toBeVisible();
  await page.getByRole("button", { name: "Accept", exact: true }).click();
  await expect(page.getByText("accepted", { exact: true })).toBeVisible();
});

test("post and block replies retain their target after reload", async ({ page }) => {
  const title = `Reply validation ${Date.now()}`;
  const parentText = "Primary evidence from the checkout logs.";
  const replyText = "This block confirms the same failure signature.";

  await page.getByLabel("Incident title").fill(title);
  await page.getByRole("button", { name: "Create incident" }).click();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();
  await page.getByPlaceholder("Share an update…").fill(parentText);
  await page.getByRole("button", { name: "Post update" }).click();

  const parent = page.locator("article.post", { hasText: parentText });
  await parent.getByRole("button", { name: "Reply to block" }).click();
  await expect(page.getByText("Replying to block by alice")).toBeVisible();
  await page.getByPlaceholder("Share an update…").fill(replyText);

  const responsePromise = page.waitForResponse(response =>
    response.request().method() === "POST" && response.url().endsWith("/posts")
  );
  await page.getByRole("button", { name: "Post update" }).click();
  const response = await responsePromise;
  expect(response.status()).toBe(201);
  const createdReply = await response.json();
  expect(createdReply.replyToPostId).toBeTruthy();
  expect(createdReply.replyToBlockId).toBeTruthy();

  const reply = page.locator("article.post-reply", { hasText: replyText });
  await expect(reply).toContainText(`Reply to block by alice: ${parentText}`);

  await page.reload();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();
  await expect(page.locator("article.post-reply", { hasText: replyText }))
    .toContainText(`Reply to block by alice: ${parentText}`);
});

test("the top-right identity control is explicitly development-only", async ({ page }) => {
  await expect(page.getByLabel("Development principal")).toHaveValue("alice");
  await expect(page.getByLabel("Development principal"))
    .toHaveAttribute("title", /Temporary local identity.*not a user profile field/);
});

test("owners can review, change, and revoke incident membership", async ({ page }) => {
  const title = `Membership validation ${Date.now()}`;
  await page.getByLabel("Incident title").fill(title);
  await page.getByRole("button", { name: "Create incident" }).click();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();

  await page.getByLabel("New participant").fill("bob");
  await page.getByLabel("Participant role").selectOption("editor");
  await page.getByRole("button", { name: "Add participant" }).click();

  const bob = page.locator(".member-row", { hasText: "bob" });
  await expect(bob).toContainText("editor · active");
  await bob.getByRole("button", { name: "Make viewer" }).click();
  await expect(bob).toContainText("viewer · active");
  await bob.getByRole("button", { name: "Revoke" }).click();
  await expect(bob).toContainText("viewer · revoked");
  await expect(bob.getByRole("button", { name: "Revoke" })).toHaveCount(0);

  await page.getByLabel("Development principal").fill("bob");
  await page.getByRole("button", { name: "Switch" }).click();
  await expect(page.getByRole("heading", { name: title })).toHaveCount(0);
  await expect(page.getByRole("button", { name: `unclassified ${title}` })).toHaveCount(0);
});
