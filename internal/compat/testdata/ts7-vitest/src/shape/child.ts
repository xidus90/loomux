import { Base } from "./base.js";

export class Child extends Base {
  override area(): number {
    return super.area() * 2;
  }
}
