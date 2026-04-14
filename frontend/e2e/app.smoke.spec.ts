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

test("toggles oil network visibility from the live top-bar menu", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Hide Oil Network").click();

  await page.getByRole("button", { name: "MENU" }).click();
  await expect(page.getByText("Show Oil Network")).toBeVisible();
});

test("opens the live scenario modal and shows harness scenarios", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Scenario").click();

  await expect(page.getByText("Proving Ground: Package Strait Regional Control")).toBeVisible();
  await expect(page.getByText("Default Scenario")).toBeVisible();
});

test("filters scenarios in the live scenario modal", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Scenario").click();

  const search = page.getByRole("searchbox", { name: "Search scenarios" });
  await search.fill("israel");

  await expect(page.getByText("Israel Air Defense Drill")).toBeVisible();
  await expect(page.getByText("Default Scenario")).toHaveCount(0);

  await search.fill("no-match");
  await expect(page.getByText("No scenarios match that search.")).toBeVisible();
});

test("switches from the live sim shell into the scenario editor", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Editor").click();

  await expect(page.getByText("Scenario Editor")).toBeVisible();
  await expect(page.getByText("Placed Units")).toBeVisible();
  await expect(page.getByText("Definitions")).toBeVisible();
});

test("edits scenario metadata and filters the unit palette in the live editor", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "MENU" }).click();
  await page.getByText("Open Editor").click();

  const nameField = page.locator(".editor-panel-left .field").filter({ hasText: "Name" }).locator("input");
  const descriptionField = page.locator(".editor-panel-left .field").filter({ hasText: "Description" }).locator("textarea");
  const authorField = page.locator(".editor-panel-left .field").filter({ hasText: "Author" }).locator("input");

  await nameField.fill("Harness Edited Scenario");
  await descriptionField.fill("Updated by real-browser test");
  await authorField.fill("Playwright");

  await expect(nameField).toHaveValue("Harness Edited Scenario");
  await expect(descriptionField).toHaveValue("Updated by real-browser test");
  await expect(authorField).toHaveValue("Playwright");

  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Saved.")).toBeVisible();

  const palette = page.locator(".palette-root");
  await palette.locator("select.palette-country-select").selectOption("ISR");
  await expect(palette.getByText("F-35I - F-35I Adir")).toBeVisible();
  await expect(palette.getByText("KC-46 - KC-46 Pegasus")).toHaveCount(0);

  const paletteSearch = palette.getByPlaceholder("Search units, types, countries...");
  await paletteSearch.fill("kc-46");
  await expect(palette.getByText("No units match the current country and search filter.")).toBeVisible();

  await palette.locator("select.palette-country-select").selectOption("USA");
  await expect(palette.getByText("KC-46 - KC-46 Pegasus")).toBeVisible();
});
