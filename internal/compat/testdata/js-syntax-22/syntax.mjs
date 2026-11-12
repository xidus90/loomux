import config from "./config.json" with { type: "json" };

const capital = /[\p{L}--[a-z]]/v;

export function startsUpper(text) {
  return capital.test(text[0]);
}

export function check() {
  return startsUpper(config.name);
}
