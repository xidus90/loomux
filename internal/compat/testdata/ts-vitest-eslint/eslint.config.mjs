import { defineConfig } from "eslint/config";
import tseslint from "typescript-eslint";

export default defineConfig({ ignores: ["coverage/"] }, tseslint.configs.recommended);
