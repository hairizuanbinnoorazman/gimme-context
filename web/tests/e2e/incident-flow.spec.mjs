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
  await expect(page.getByRole("button", { name: "New incident" })).toBeEnabled();
});

test.afterEach(async ({ page }) => {
  expect(errorsByPage.get(page), "browser console errors").toEqual([]);
});

test("a human creates an incident and posts an investigation update", async ({ page }) => {
  const title = `Checkout latency ${Date.now()}`;
  await page.getByLabel("New channel title").fill(title);
  await page.getByLabel("Incident severity").selectOption("SEV-2");
  await page.getByLabel("Incident scope").fill("checkout, production");
  await page.getByRole("button", { name: "New incident" }).click();

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
  await page.getByLabel("New channel title").fill(title);
  await page.getByRole("button", { name: "New incident" }).click();
  await page.getByRole("button", { name: `unclassified ${title}` }).click();

  await page.getByLabel("Coordination type").selectOption("decision");
  await page.getByPlaceholder("Statement, task, or question").fill("Roll back the payments deployment.");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByText("Roll back the payments deployment.")).toBeVisible();
  await page.getByRole("button", { name: "Accept", exact: true }).click();
  await expect(page.getByText("accepted", { exact: true })).toBeVisible();
});
