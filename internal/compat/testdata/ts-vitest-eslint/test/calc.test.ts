import { expect, it } from "vitest";
import { twice } from "../src/calc.js";

it("doubles the area", () => {
  expect(twice(2)).toBe(4);
});
