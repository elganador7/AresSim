import { expect, test } from "@playwright/test";
import { installAppHarness } from "./support/appHarness";

test.beforeEach(async ({ page }) => {
  await installAppHarness(page);
});

test("boots the live app shell and shows oil network state", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByText("No Scenario")).toBeVisible();
  await page.getByRole("button", { name: "MENU" }).click();
  await expect(page.getByText("Focus Oil Network (2 nodes)")).toBeVisible();
  await expect(page.getByText("Hide Oil Network")).toBeVisible();
});

test("opens the live scenario modal and shows harness scenarios", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Scenario").click();

  await expect(page.getByText("Proving Ground: Package Strait Regional Control")).toBeVisible();
  await expect(page.getByText("Default Scenario")).toBeVisible();
});

test("switches from the live sim shell into the scenario editor", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Editor").click();

  await expect(page.getByText("Scenario Editor")).toBeVisible();
  await expect(page.getByText("Placed Units")).toBeVisible();
  await expect(page.getByText("Definitions")).toBeVisible();
});
