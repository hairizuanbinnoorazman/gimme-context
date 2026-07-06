import { test, expect } from "@playwright/test";

test("administration exposes capability status and working configuration flows", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Workspace administration" }).click();

  await expect(page.getByRole("heading", { name: "Workspace administration" })).toBeVisible();
  await expect(page.getByText("Development identity", { exact: true })).toBeVisible();
  await expect(page.getByText("Not implemented", { exact: true })).toHaveCount(2);

  await page.getByLabel("Name", { exact: true }).fill("Codex incident synthesizer");
  await page.getByLabel("Purpose", { exact: true }).fill("Summarize visible incident evidence");
  await page.getByRole("button", { name: "Approve agent" }).click();
  await expect(page.getByText("Codex incident synthesizer", { exact: true })).toBeVisible();

  await page.getByLabel("Template name").fill("Production incident");
  await page.getByLabel("Default scope").fill("checkout, production");
  await page.getByRole("button", { name: "Publish version 1" }).click();
  await expect(page.getByText("Production incident", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Refresh events" }).click();
  await expect(page.getByText("agent.approved", { exact: true })).toBeVisible();
  await expect(page.getByText("template.version_created", { exact: true })).toBeVisible();
});
