import assert from "node:assert/strict";
import test from "node:test";
import { versionsMatch } from "./version.js";

test("0.x matches on minor", () => {
  assert.equal(versionsMatch("0.1.0-dev", "0.1.0"), true);
  assert.equal(versionsMatch("0.2.0", "0.1.0"), false);
});

test("1.x matches on major", () => {
  assert.equal(versionsMatch("1.4.0", "1.0.0"), true);
  assert.equal(versionsMatch("2.0.0", "1.0.0"), false);
});
