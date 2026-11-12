import config from "./config.json" with { type: "json" };

const capital = /[\p{L}--[a-z]]/v;
const name = /(?i:loom)ux/;
const year = /(?<year>\d{4})-\d{2}|\d{2}-(?<year>\d{4})/;

export function open() {
  return { [Symbol.dispose]() {} };
}

export function check(text) {
  using handle = open();
  return capital.test(text[0]) && name.test(text) && year.exec(text)?.groups?.year === config.year;
}
