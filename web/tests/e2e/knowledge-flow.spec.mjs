import { expect, test } from "@playwright/test";

test("a permanent channel has a distinct creation flow and responds to posts", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Permanent", exact: true }).click();
  await expect(page.getByLabel("Permanent channel title")).toBeVisible();
  await expect(page.getByLabel("Incident severity")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create permanent channel" })).toBeDisabled();

  const title = `Checkout operations ${Date.now()}`;
  await page.getByLabel("Permanent channel title").fill(title);
  await page.getByRole("button", { name: "Create permanent channel" }).click();
  await page.getByRole("button", { name: title, exact: true }).click();

  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  await expect(page.getByText("Permanent channel", { exact: true })).toBeVisible();
  await page.getByPlaceholder("Share an update…").fill("Checkout rollback steps are documented here.");
  await page.getByRole("button", { name: "Post update" }).click();
  await expect(page.getByText("Checkout rollback steps are documented here.")).toBeVisible();
  await expect(page.getByPlaceholder("Share an update…")).toHaveValue("");

  const parent = page.locator("article.post", { hasText: "Checkout rollback steps are documented here." });
  await parent.getByRole("button", { name: "Reply to post" }).click();
  await expect(page.getByText("Replying to post by alice")).toBeVisible();
  await page.getByPlaceholder("Share an update…").fill("The rollback owner has reviewed these steps.");
  await page.getByRole("button", { name: "Post update" }).click();
  await expect(page.locator("article.post-reply", { hasText: "The rollback owner has reviewed these steps." }))
    .toContainText("Reply to alice: Checkout rollback steps are documented here.");
});
