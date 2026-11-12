interface Sized {
  size(): number;
}

type Pair<T> = readonly [T, T];

export class Box implements Sized {
  width = 2;

  size(): number {
    return this.width * 2;
  }
}

export function pair<const T>(value: T): Pair<T> {
  return [value, value];
}

export function measure(): number {
  using handle = { [Symbol.dispose]() {} };
  const box = new Box() satisfies Sized;
  return box.size() + pair(1).length;
}
