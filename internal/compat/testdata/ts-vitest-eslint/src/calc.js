import { Child } from "./shape/child.ts";

export function twice(value) {
  return new Child().area() * value;
}
